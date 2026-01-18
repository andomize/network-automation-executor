package controller

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/andomize/network-automation-executor/internal/adapters/filestorage"
	"github.com/andomize/network-automation-executor/internal/adapters/jsontask"
	"github.com/andomize/network-automation-executor/internal/adapters/logger"
	"github.com/andomize/network-automation-executor/internal/core/domains"
	"github.com/andomize/network-automation-executor/internal/core/ports"
	"github.com/andomize/network-automation-executor/internal/core/services/spawner"
)

type Artefacts map[string]string

type NamedTask struct {
	Status string
	Output string
}

type Controller struct {

	// Экземпляр GExpect для реализации Spawn
	Connection *spawner.Connection

	// Шаблон, описывающий все поля задания
	Task domains.TaskPattern

	// Директория в файловой системе для хранения выводов
	OutputStorage *filestorage.FileStorage

	// Данные именованных заданий (С полем <name>)
	Names map[string]*NamedTask

	// Данные для перехода от задания к заданию
	NextTaskName string

	// Данные, доступные для подстановки в задания (переменные)
	Variables Artefacts

	// Путь к файлу задания
	TaskPath string
}

/*
 * NewController
 *
 * Создаёт новый экземпляр Controller
 */
func NewController(taskPath, outputDirectory, user, pass string) (*Controller, error) {

	logger.DEBUG("CTRL_NEW: Start creating new controller with task path: '" + taskPath +
		"' and outputDirectory: '" + outputDirectory + "'")

	// Контроллер пути с выходными файлами задания
	outputStorage := filestorage.NewFileStorage(outputDirectory)

	// Читаем задание из файловой системы в память контроллера
	fsysTask, fsysTaskReadError := jsontask.Read(taskPath)

	if fsysTaskReadError != nil {
		logger.ERROR("CTRL_NEW: Cannot read task file '" + taskPath +
			"' by reason: " + fsysTaskReadError.Error())
		return nil, fsysTaskReadError
	}

	// Создаём экземпляр Controller
	controller := &Controller{
		Task:          *fsysTask,
		OutputStorage: outputStorage,
		Names:         map[string]*NamedTask{},
		NextTaskName:  "",
		Variables:     Artefacts{},
		TaskPath:      taskPath,
	}

	if connError := controller.connect(fsysTask.Host, user, pass); connError != nil {
		controller.ExitError(connError.Error())
	}

	// Базовая проверка файла задания
	if len(controller.Task.Host) <= 0 {
		controller.ExitError(ports.ERROR_SYNTAX_NO_HOST)
	}
	if controller.Task.Tasks == nil || len(*controller.Task.Tasks) <= 0 {
		controller.ExitError(ports.ERROR_SYNTAX_NO_TASKS)
	}

	// Добавляем системные переменные (предопределяются по умолчанию)
	controller.Variables["host"] = controller.Task.Host
	controller.Variables["date"] = time.Now().Format("2006-01-02")
	controller.Variables["time"] = time.Now().Format("15-04-05")
	controller.Variables["vendor"] = controller.Connection.Prompt.Vendor
	controller.Variables["prompt"] = controller.Connection.Prompt.Name

	// Актуализируем информацию о задании на основе полученных данных
	controller.Task.Vendor = controller.Connection.Prompt.Vendor

	// Переносим переменные из задания (если они есть) в память контроллера
	for index, value := range controller.Task.Variables {
		logger.DEBUG("CTRL_NEW: Adding new variable '" + index + "' = '" + value + "'")
		controller.Variables[index] = value
	}

	return controller, nil
}

/*
 * Controller.ExitSuccess
 *
 * Записать успешный статус в задание, сохранить файл и выйти из программы
 */
func (c *Controller) ExitSuccess() {
	logger.DEBUG("CTRL_EXIT_SUCCESS: Set new status for task: '" + ports.PIPE_STATUS_SUCCESS +
		"' and program exiting")

	// Устанавливаем новый статус для задания - УСПЕШНО
	c.Task.Status = ports.PIPE_STATUS_SUCCESS
	// Сохранение задания, закрытие служб, выход
	c.Save()
	c.Close()
	os.Exit(0)
}

/*
 * Controller.ExitError
 *
 * Записать ошибку в задание, сохранить файл и выйти из программы
 */
func (c *Controller) ExitError(errorCode string) {
	logger.DEBUG("CTRL_EXIT_ERROR: Set new error: '" + errorCode + "' and exiting")

	c.Task.Error = errorCode
	c.Task.Status = ports.PIPE_STATUS_FAIL
	c.Save()
	c.Close()
	log.Fatal(errors.New(errorCode))
}

/*
 * Controller.connect
 *
 * Подключение к удалённому устройству
 */
func (c *Controller) connect(host, user, pass string) error {
	// Открываем сессию с удалённым хостом. Процесс использует модуль GExpect
	// для подключения к хосту, используя протоколы SSH1, SSH, Telnet
	connection, connectionError := spawner.NewConnection(host, user, pass)

	if connectionError != nil {
		logger.ERROR("CTRL_NEW: Connection to host '" + host + "' failed " +
			"by reason: " + connectionError.Error())
		return connectionError
	}

	c.Connection = connection
	return nil
}

/*
 * Controller.SaveOutput
 *
 * Сохранить вывод в файл
 */
func (c *Controller) SaveOutput(output, filename string) error {
	logger.DEBUG("CTRL_SEND: Starting save output file: " + filename + "'")
	return c.OutputStorage.Save([]byte(output), filename)
}

/*
 * Controller.Save
 *
 * Сохранить файл задания
 */
func (c *Controller) Save() {
	logger.DEBUG("CTRL_SAVE: Starting saving task file '" + c.TaskPath + "'")
	jsontask.Write(c.TaskPath, c.Task)
}

/*
 * Controller.Close
 *
 * Закрыть все службы контроллера заданий
 */
func (c *Controller) Close() {
	logger.DEBUG("CTRL_CLOSE: Closing task controller")
	if c.Connection != nil {
		c.Connection.Close()
	}
}
