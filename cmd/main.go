package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/andomize/network-automation-executor/internal/adapters/environment"
	"github.com/andomize/network-automation-executor/internal/adapters/logger"
	"github.com/andomize/network-automation-executor/internal/core/domains"
	"github.com/andomize/network-automation-executor/internal/core/ports"
	"github.com/andomize/network-automation-executor/internal/core/services/controller"
)

func main() {

	taskPath, outputDirectory, flagsError := GetFlags()
	logger.Must(flagsError, "Arguments is wrong")

	username := environment.Get("CLI_USERNAME", "", true)
	password := environment.Get("CLI_PASSWORD", "", true)

	// Читаем содержимое файла задания и на основе него создаём контроллер
	controller, controllerError := controller.NewController(
		taskPath, outputDirectory, username, password)
	logger.Must(controllerError, "Cannot create task controller")
	defer controller.ExitSuccess()

	logger.INFO("Connection to host '" + controller.Task.Host + "' successful")

	// Запускаем поочерёдное выполнение заданий
	Run(controller, controller.Task.Tasks, controller.Variables, 0)

	// Если необходимо тестирование задания, то выполняем тестирование
	if autotestsError := controller.Autotests(); autotestsError != nil {
		controller.ExitError(autotestsError.Error())
	}
}

/*
 * Run
 *
 * Передаём на исполнение все задания текущего уровня
 * Задания будут передататься рекурсивно до тех пор, пока не будет достугнут
 * конечный уровень вложенности
 */
func Run(
	ctrl *controller.Controller,
	tasks *[]domains.Task,
	variables controller.Artefacts,
	depthLevel int,
) {

	logger.DEBUG(fmt.Sprintf("RUN: Starting tasks execution, depth: %v, taskCount: %v",
		depthLevel, len(*tasks)))

	// Выполняем поочерёдно все задания текущего уровня
	// Текущим уровнем может быть и корневой уровень в том числе, если подзаданий не существует
	// у любого из заданий корневого уровня, то по истечению данного цикла (выполнению всех
	// подзадач текущего (корневого) уровня), будет осуществлён выход из программы
	for taskIdx, task := range *tasks {

		logger.DEBUG(fmt.Sprintf("RUN: Processing task '%v' with command '%v'", taskIdx, task.Command))

		// Компилируем задание
		//  - Вместо имён всех переменных подставляются их значения
		//  - Вычисляется Timeout для ожидания ответа на основе данных в задании
		ctrl.Compile(&task, variables)

		// Проверяем установлено ли имя следующего задания
		if len(ctrl.NextTaskName) > 0 {
			// Если установлено имя следующего задания, то все задания, не соответствующие
			// искомому - будут пропускаться
			if task.Name != ctrl.NextTaskName {
				logger.WARNING("RUN: Command: '" + task.Command + "' was skipped by GOTO action")
				ctrl.SetTaskStatus(&(*tasks)[taskIdx], ports.PIPE_STATUS_SKIPPED)
				continue
			}
			// Обнуляем имя следующего задания, т.к. оно было достигнуто
			// далее задания будут выполняться в нормальном порядке
			ctrl.NextTaskName = ""
		}

		// Проверяем что задание не выполнялось ранее
		// Если у задания в поле "status" уже есть результат выполнения, то это означает,
		// что данный файл задания уже запускался ранее и необходимо защитить удалённых хост
		// от повторного выполнения команд, если это не разрешено в явном виде
		if len(task.Status) > 0 {
			// Данное задание уже выполнялось
			// Проверим разрешено ли повторное исполнение выполнение текущего задания
			// Если задание - это подзадание n-го уровня, то ошибки быть не должно т.к.
			// для подзадач свойственнен множественный запуск
			if !task.Params.CommandRepeatAllowed && depthLevel == 0 {
				// Разрешение на повторное выполнение отсутствует и это корневое задание
				// Устанавливаем новый статус задания - SKIPPED
				// Выполняем переход к следующему заданию
				logger.WARNING("RUN: Command: '" + task.Command + "' has already been executed")
				ctrl.SetTaskStatus(&(*tasks)[taskIdx], ports.PIPE_STATUS_SKIPPED)
				continue
			} else {
				// Разрешение на повторное выполнение присутствует
				//logger.INFO("RUN: Command: '" + task.Command + "' has already been executed," +
				//	" but CommandRepeatAllowed is true, continue...")
			}
		}

		// Проверяем еть ли в задании условия и выполняются ли они
		conditionsSuccess, conditionsError := ctrl.WhenMatcher(task.When, variables)

		if conditionsError != nil {
			// Если была выявлена ошибка на логическом/программном уровне - завершнние работы
			logger.ERROR("RUN: Command '" + task.Command + "' conditions is fail, error")
			ctrl.ExitError(conditionsError.Error())

		} else {
			// Если условное выражение из задания неуспешно, то пропускаем (Условие не выполнено)
			if !conditionsSuccess {
				logger.INFO("RUN: Command '" + task.Command + "' conditions is fail, continue...")
				ctrl.SetTaskStatus(&(*tasks)[taskIdx], ports.PIPE_STATUS_SKIPPED)
				continue
			}
		}

		// Выполняем отправку команды на удалённое устройство
		output, commandSendError := ctrl.Send(&task)

		if commandSendError != nil {
			// Команда была отправлена с ошибками
			logger.DEBUG("RUN: Send command: '" + task.Command + "' failed")
			ctrl.SetTaskStatus(&(*tasks)[taskIdx], ports.PIPE_STATUS_FAIL)

			// Аргумент OnErrorContinue разрешает продолжение выполнения команд даже если
			// текущая команда была выполнена с ошибкой. Проверяем установлен ли данный флаг
			if task.Params.OnErrorContinue {

				// Команда не была выполнена - Продолжение разрешено
				logger.WARNING("RUN: Send command: '" + task.Command + "' failed," +
					" but OnErrorContinue is true, continue...")
			} else {

				// Команда не была выполнена - Продолжение недоступно - выход
				logger.ERROR(fmt.Sprintf("RUN: Send command: '%s' failed by reason: '%v'",
					task.Command, commandSendError))
				ctrl.ExitError(commandSendError.Error())
			}
		} else {
			// Команда была отправлена успешно
			logger.INFO("RUN: Send command: '" + task.Command + "' successful")
			ctrl.SetTaskStatus(&(*tasks)[taskIdx], ports.PIPE_STATUS_SUCCESS)
		}

		// Флаг OutputFile (если он не пустой) говорит о том, что необходимо выполнить сохранение
		// вывода от текущей команды в файл, имя которого указано в данной переменной
		if len(task.Params.OutputFile) > 0 && (*tasks)[taskIdx].Status == ports.PIPE_STATUS_SUCCESS {
			// Сохраняем в файл полученный вывод после выполнения команды
			savingFileError := ctrl.SaveOutput(output, task.Params.OutputFile)

			if savingFileError != nil {
				logger.ERROR("Saving output file is fail by reason: " + savingFileError.Error())
				ctrl.ExitError(savingFileError.Error())
			}

			logger.INFO("RUN: Save output to file: '" + task.Params.OutputFile + "' successful")
		}

		// Выполняем проверку на наличие параметра "Filter" в задании
		// Если данный параметр существует, то необходимо распарсить вывод, полученный
		// после отправки команды и сгенерировать соответствующие подзадания с
		// подстановкой в качестве заданных переменных (например, {{1}} или {{name}})
		// найденных в результате парсинга значений
		if len(task.Params.Filter) > 0 {

			// Распарсим вывод на основе регулярного выражения
			// task.Params.Filter - Регулярное выражение по которму будет получен срез
			//     найденных в выводе значений для дальнейшей подстановки в задания
			// task.Params.FilterExclude - Регулярное выражение по которому полученные
			//     ранее значения будут исключены из среза
			regMap, regCount, regError := ctrl.RegExpMatch(
				output, task.Params.Filter, task.Params.FilterExclude)

			// Если не удалось распарсить вывод, используя заложенное регулярное выражение,
			// то подзадания выполняться не будут, а для задания будет установлено ошибочное
			// состояние
			if regError != nil {
				logger.ERROR("RUN: Regular expression is fail by reason: " + regError.Error())
				ctrl.ExitError(regError.Error())
			}

			// Проверяем сколько элементов содержит результат парсинга вывода регулярным выражением
			// Если количество результатов нулевое, то продолжать нет смысла - следующее задание
			if regCount <= 0 {
				logger.INFO("RUN: Regular expression returns zero values, skipping...")
				ctrl.SetTaskStatus(&(*tasks)[taskIdx], ports.PIPE_STATUS_SKIPPED)
				continue
			}

			// В зависимоти от количества найденных регулярным выражением значений из вывода,
			// необходимо запустить подзадание соответтсвующее количество раз, подставляя в
			// набор артефактов новые значения, которые были получены по регулярному выражению
			//
			// Изначальный формат:
			//
			//	|-------------------------------------------|
			//	| Команда: show vdc detail                  |
			//	|-------------------------------------------|
			//	    |
			//	    |    |----------------------------------|
			//	    |--> | Команда: switchto vdc {{vdc}}    |
			//	    |    |----------------------------------|
			//	    |--> | Команда: show vrf detail         |
			//	    |    |----------------------------------|
			//	             |
			//	             |    |-------------------------------------|
			//	             |    | Команда: show ip route vrf {{vrf}}  |
			//	             |    |-------------------------------------|
			//	             |--> | Команда: show ip arp vrf {{vrf}}    |
			//	                  |-------------------------------------|
			//
			// Сформированные задания:
			//
			//	 artefacts = {vdc: [L3-CORE, AGG]}
			//	|-------------------------------------------|
			//	| Команда: show vdc detail                  |
			//	|-------------------------------------------|
			//	    |
			//	    |     artefacts = {vdc : L3-CORE, vrf: [mgmt]}
			//	    |    |-----------------------------------|
			//	    |--> | Команда: switchto vdc L3-CORE     |
			//	    |    |-----------------------------------|
			//	    |
			//	    |     artefacts = {vdc : L3-CORE, vrf: [big-data, inside]}
			//	    |    |-----------------------------------|
			//	    |--> | Команда: show vrf detail          |
			//	    |    |-----------------------------------|
			//	    |        |
			//	    |        |     artefacts = {vdc: L3-CORE, vrf: big-data}
			//	    |        |    |------------------------------------------|
			//	    |        |    | Команда: show ip route vrf big-data      |
			//	    |        |    |------------------------------------------|
			//	    |        |--> | Команда: show ip arp vrf big-data        |
			//	    |        |    |------------------------------------------|
			//	    |        |
			//	    |        |     artefacts = {vdc: L3-CORE, vrf: inside}
			//	    |        |    |------------------------------------------|
			//	    |        |    | Команда: show ip route vrf inside        |
			//	    |        |    |------------------------------------------|
			//	    |        |--> | Команда: show ip arp vrf inside          |
			//	    |             |------------------------------------------|
			//	    |
			//	    |     artefacts = {vdc : AGG, vrf: [mgmt]}
			//	    |    |-----------------------------------|
			//	    |--> | Команда: switchto vdc AGG         |
			//	    |    |-----------------------------------|
			//	    |
			//	    |     artefacts = {vdc : AGG, vrf: [mgmt]}
			//	    |    |-----------------------------------|
			//	    |--> | Команда: show vrf detail          |
			//	         |-----------------------------------|
			//	             |
			//	             |     artefacts = {vdc: AGG, vrf: mgmt}
			//	             |    |------------------------------------------|
			//	             |    | Команда: show ip route vrf mgmt          |
			//	             |    |------------------------------------------|
			//	             |--> | Команда: show ip arp vrf mgmt            |
			//	                  |------------------------------------------|
			//
			if task.Tasks != nil {

				logger.DEBUG(fmt.Sprintf("RUN: Task has subtasks and regular expression"+
					" return '%d' values in any groups, map: '%v', default artefacts is: '%v'",
					regCount, regMap, variables))

				// В зависимоти от числа полученных артефактов, подзадания нужно запустить
				// в аналогичном объёме, подставляя в каждое подзадание новое значение
				// артефакта из массива артефактов
				for subTaskOrder := 0; subTaskOrder < regCount; subTaskOrder++ {

					// Определим новую переменную, которая будет хранить в себе артефакты
					// только для конкретного подзадания. Это используется для того, что бы
					// данные из подзаданий не могли переноситься между смежными подзаданиями
					// или даже в следующее родительское задание
					var subTaskArtefacts = map[string]string{}
					for index, value := range variables {
						subTaskArtefacts[index] = value
					}

					// Добавляем новые артефакты для следующего задания
					// Если ранее существовали артефакты с идентичными идентификаторами,
					// то такие артефакты будут перезаписаны на новое значение, при чём
					// во время возврата к предыдущему заданию будут доступны прежние значения
					for index, values := range regMap {
						subTaskArtefacts[index] = values[subTaskOrder]

						logger.DEBUG(fmt.Sprintf("RUN: Set new value for artefact id"+
							" '%s' = '%s'", index, values[subTaskOrder]))
					}

					logger.DEBUG("RUN: Starting new subtask using artefacts: '" +
						fmt.Sprint(subTaskArtefacts))

					// Рекурсивно апускаем выполнение следующего задания
					Run(ctrl, (*tasks)[taskIdx].Tasks, subTaskArtefacts, depthLevel+1)
				}
			} else {
				logger.WARNING("RUN: Task contains regular expression," +
					" but not contains subtasks, skipping...")
				ctrl.SetTaskStatus(&(*tasks)[taskIdx], ports.PIPE_STATUS_SKIPPED)
			}
		}
	}
}

/*
 * GetFlags
 *
 * Получить список всех флагов, с которыми была запущена программ
 */
func GetFlags() (string, string, error) {

	var taskArg string
	var outputArg string
	var debugArg bool
	var version bool

	flag.StringVar(&taskArg, "t", "", "Path to task file")
	flag.StringVar(&outputArg, "o", "", "Path to output directory")
	flag.BoolVar(&debugArg, "d", false, "Debug mode")
	flag.BoolVar(&version, "version", false, "Show program version")

	// After parsing, the arguments following the flags are available
	// as the slice flag.Args() or individually as flag.Arg(i).
	// The arguments are indexed from 0 through flag.NArg()-1.
	flag.Parse()

	// Если указан флаг для запроса версии, то выводим её
	if version {
		fmt.Println(ports.VERSION)
		os.Exit(0)
	}

	// Если указан флаг Debug, то включаем Debug режим
	if debugArg {
		logger.ModuleEnableDebug()
	}

	// Выполняем проверку обязательных флагов
	if len(taskArg) <= 0 || len(outputArg) <= 0 {
		return "", "",
			errors.New("Required flags not found\n" +
				"-t (TASK FILE):\n" +
				"    /var/tasks/TASK.json\n" +
				"    tasks/TASK.json\n" +
				"    TASK.json\n\n" +
				"-o (OUTPUT DIRECTORY):\n" +
				"    /var/outputs/\n" +
				"    outputs/\n" +
				"    outputs\n")
	}

	// Transform directories/filenames from flags to absolute paths
	// Normalize the path to the task file
	taskdir, taskerr := filepath.Abs(filepath.Dir(taskArg))
	taskfile := filepath.Base(taskArg)
	taskPath := filepath.Join(taskdir, taskfile)

	// Normalize the path to the log file
	outputDirectory, logerr := filepath.Abs(outputArg)

	logger.DEBUG("Path to task: \"" + taskPath + "\"")
	logger.DEBUG("Path to logs: \"" + outputDirectory + "\"")

	// Verifying that absolute paths successful created
	if taskerr != nil || logerr != nil {
		return "", "", errors.New("Path(s) are unacceptable")
	}

	return taskPath, outputDirectory, nil
}
