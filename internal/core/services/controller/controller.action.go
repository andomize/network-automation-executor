package controller

import (
	"github.com/andomize/network-automation-executor/internal/adapters/logger"
	"github.com/andomize/network-automation-executor/internal/core/domains"
	"github.com/andomize/network-automation-executor/internal/core/ports"
)

/*
 * Controller.Compile
 *
 * Сформировать поля задания на основе имеющихся данных
 */
func (c *Controller) Compile(task *domains.Task, vars Artefacts) {
	logger.DEBUG("CTRL_CLOSE: Starting compile task pattern")

	// Конвертируем переменные в команде на значения переменных
	command, commandSubError := c.RegExpConstructor(task.Command, vars)
	if commandSubError != nil {
		logger.ERROR("CTRL_COMPILE: Fail to construct command" +
			"by reason: " + commandSubError.Error())
		c.ExitError(commandSubError.Error())
	}

	// Конвертируем переменные в имени файла вывода
	outputFile, outputFileError := c.RegExpConstructor(task.Params.OutputFile, vars)
	if outputFileError != nil {
		logger.ERROR("CTRL_COMPILE: Fail to construct output filename" +
			"by reason: " + outputFileError.Error())
		c.ExitError(outputFileError.Error())
	}

	// Проверяем установлен ли специфичный Timeout для задания
	if task.Params.Timeout <= 0 {
		// Если Timeout не установлен, то устанавливаем глобальный или системный
		task.Params.Timeout = c.GetDefaultTimeout()
	}

	task.Command = command
	task.Params.OutputFile = outputFile
}

/*
 * Controller.Send
 *
 * Отправить команду на хост
 */
func (c *Controller) Send(task *domains.Task) (string, error) {
	logger.DEBUG("CTRL_SEND: Starting send task command: '" + task.Command + "'")

	commandSendOutput, commandSendError := c.Connection.Send(
		task.Command, task.Params.Timeout, task.Params.PromptChangeAllowed)

	if commandSendError == nil {
		// Установим новое значение переменной prompt
		c.Variables["prompt"] = c.Connection.Prompt.Name
	}

	// Проверяем присутствует ли поле <name> в теле задания
	// Если данное поле присутствует, то есть вероятность, что информация из данного задания
	// будет принимать участие при обработке условного оператора в следующем задании
	if len(task.Name) > 0 {
		logger.DEBUG("CTRL_SEND: Enriching a named task '" + task.Name + "' with output")
		if c.Names[task.Name] != nil {
			c.Names[task.Name].Output = commandSendOutput
		} else {
			c.Names[task.Name] = &NamedTask{Output: commandSendOutput}
		}
	}

	return commandSendOutput, commandSendError
}

/*
 * Controller.SetTaskStatus
 *
 * Установить новый статус для задания
 */
func (c *Controller) SetTaskStatus(task *domains.Task, status string) {
	task.Status = status

	// Проверяем присутствует ли поле <name> в теле задания
	// Если данное поле присутствует, то есть вероятность, что информация из данного задания
	// будет принимать участие при обработке условного оператора в следующем задании
	if len(task.Name) > 0 {
		logger.DEBUG("CTRL_SEND: Enriching a named task '" + task.Name + "' with status '" + status + "'")
		if c.Names[task.Name] != nil {
			c.Names[task.Name].Status = status
		} else {
			c.Names[task.Name] = &NamedTask{Status: status}
		}
	}
}

/*
 * Controller.GetDefaultTimeout
 *
 * Чтение глобального значения Timeout для ожидания выполнения задания
 * Если глобальное значение Timeout отсутствует, то берём системное время
 */
func (c *Controller) GetDefaultTimeout() int {
	if c.Task.Settings != nil {
		if c.Task.Settings.Timeout != 0 {
			return c.Task.Settings.Timeout
		}
	}

	return ports.SPAWN_TIMEOUT_SYSTEM
}
