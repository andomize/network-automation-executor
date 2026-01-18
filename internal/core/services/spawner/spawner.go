package spawner

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/andomize/network-automation-executor/internal/adapters/logger"
	"github.com/andomize/network-automation-executor/internal/core/ports"
)

type Connection struct {

	// Содержит экземпляр Spawn-сессии
	spawn *Spawn

	// Содержит экземпляр текущего обнаруженного Prompt'a
	Prompt *Prompt

	// Содержит вывод с устройства при первоначальном подключении
	connectOutput string
}

/*
 * NewConnection
 *
 * Базовый метод для подключения к удалённому устройству
 * Выполняется попытка установить удалённое соединение посредством следующих
 * возможных утилит: ssh1, ssh, telnet
 */
func NewConnection(host, username, password string) (*Connection, error) {

	// Предопределим команду вызова утилиты SSH
	// ssh -o connecttimeout=20 -o StrictHostKeyChecking=no ... user@host
	// Предопределим все аргументы для вызова утилиты SSH
	var ssh_KexAlgorithms = "-o KexAlgorithms=+diffie-hellman-group1-sha1," +
		"diffie-hellman-group14-sha1,diffie-hellman-group14-sha256," +
		"diffie-hellman-group16-sha512,diffie-hellman-group-exchange-sha1," +
		"diffie-hellman-group-exchange-sha256,ecdh-sha2-nistp256," +
		"ecdh-sha2-nistp384,ecdh-sha2-nistp521,curve25519-sha256"
	var ssh_Ciphers = "-o Ciphers=+aes128-cbc,3des-cbc,aes192-cbc,aes256-cbc"
	var ssh_HostKeyAlgorithms = "-o HostKeyAlgorithms=+ssh-dss,ssh-rsa"
	var ssh_command = "ssh -o connecttimeout=20 -o StrictHostKeyChecking=no " +
		ssh_KexAlgorithms + " " + ssh_HostKeyAlgorithms + " " + ssh_Ciphers +
		" " + username + "@" + host

	// Предопределим команду вызова утилиты SSH1
	// ssh1 -o connecttimeout=20 -o StrictHostKeyChecking=no user@host
	var ssh1_command = "ssh1 -o connecttimeout=20 -o StrictHostKeyChecking=no" +
		" " + username + "@" + host

	// Предопределим команду вызова утилиты Telnet
	// telnet -l user host
	var telnet_command = "telnet -l " + username + " " + host

	// ПОПЫТКА 1. Подключение с использованием SSH1
	spawnSSH1, outputSSH1, errorSSH1 := NewSpawn(username, password, ssh1_command)
	if errorSSH1 == nil {
		// Успешное подключение с испольованием протокола SSH1
		logger.DEBUG("CONN_NEW: Connection using SSH1 successful")

		connection := Connection{
			spawn:         spawnSSH1,
			connectOutput: outputSSH1,
		}
		return &connection, connection.PromptDefine()
	}

	// ПОПЫТКА 2. Подключение с использованием SSH
	spawnSSH, outputSSH, errorSSH := NewSpawn(username, password, ssh_command)
	if errorSSH == nil {
		// Успешное подключение с испольованием протокола SSH
		logger.DEBUG("CONN_NEW: Connection using SSH successful")

		connection := Connection{
			spawn:         spawnSSH,
			connectOutput: outputSSH,
		}
		return &connection, connection.PromptDefine()
	}

	// ПОПЫТКА 3. Подключение с использованием Telnet
	spawnTelnet, outputTelnet, errorTelnet := NewSpawn(username, password, telnet_command)
	if errorTelnet == nil {
		// Успешное подключение с испольованием протокола Telnet
		logger.DEBUG("CONN_NEW: Connect using Telnet successful")

		connection := Connection{
			spawn:         spawnTelnet,
			connectOutput: outputTelnet,
		}
		return &connection, connection.PromptDefine()
	}

	// Если ни одна из попыток подключиться не была успешной,
	// то возвращаем ошибку с текстом из первой попытки подключения
	// Заранее проверяем что содержится корректная ошибка
	if errorSSH1 != nil && errorSSH1.Error() != ports.ERROR_INTERNAL_EXEC {
		return nil, errorSSH1
	}
	if errorSSH != nil && errorSSH.Error() != ports.ERROR_INTERNAL_EXEC {
		return nil, errorSSH
	}
	if errorTelnet != nil && errorTelnet.Error() != ports.ERROR_INTERNAL_EXEC {
		return nil, errorTelnet
	}

	return nil, errors.New(ports.ERROR_CONN_NO_AVAILABLE_METHOD)
}

/*
 * Connection.Send
 *
 * <command> will be sent to device and if output contains
 * 	<exec> string function will return nil and it's mean
 * 	that command successful send to device..
 * If remote device after send command do not return exec
 * 	string it's mean that command failed..
 */
func (c *Connection) Send(command string, timeout int, promptChangeAllowed bool) (string, error) {

	// Сохраняем текущий Prompt для дальнейшего сравнения
	currentPrompt := c.Prompt
	nextPrompt := c.Prompt

	// Если разрешена смена Prompt для текущей выполняемой команды,
	// то нам необходимо установить новый захватываемый Prompt,
	// а именно - универсальный, т.к. новый Prompt не будет захвачен
	if promptChangeAllowed {
		nextPrompt = &PromptUniversal
	}

	// Выполняем отправку команды на удалённое устройство
	// Передаём Prompt, который ожидаем увидеть после выполнения команды
	output, sendError := c.spawn.SendString(command, timeout, nextPrompt)

	// Удаляем служебные символы и лишние пробелы по краям вывода
	// p.s. это только для отдачи запросчику (не участвует в логике)
	output = strings.Trim(output, "\r\n ")
	// Удаляем лишние слежубные символы - {20 08}, {32, 08} (Нужно для F5)
	output = strings.Replace(output, string([]byte{20, 8}), "", -1)
	output = strings.Replace(output, string([]byte{32, 8}), "", -1)
	// Удаляем последнюю строку вывода т.е. Prompt
	output = strings.Replace(output, output[strings.LastIndex(output, "\n")+1:], "", 1)

	// Если команда была отправлена с ошибками, то выходим из метода без
	// дальнейшего определения Prompt
	if sendError != nil {
		logger.DEBUG("CONN_SEND: Command: '" + command +
			"' sending failed by reason: " + sendError.Error())
		return output, sendError
	}

	// Повторно идентифицируем Prompt после успешного выполнения команды
	promptDefineError := c.PromptDefine()

	// Проверяем что Prompt устройства был корректно определён
	if promptDefineError != nil {
		logger.DEBUG("CONN_SEND: After send command: '" + command +
			"' prompt is undefined by reason: " + promptDefineError.Error())
		return output, promptDefineError
	}

	// Новый Prompt устройства был успешно захвачен
	// Если текущий захваченный Prompt отличается от прежнего
	// и не установлен разрешающий флаг смены Prompt, то вызываем ошибку
	if c.Prompt.Name != currentPrompt.Name && !promptChangeAllowed {
		logger.DEBUG("CONN_SEND: After send command: '" + command +
			"' prompt has been changed, but its not allowed!")
		return output, errors.New(ports.ERROR_PROMPT_CHANGED)
	}

	return output, nil
}

/*
 * Connection.CiscoMenuAction
 *
 * Осуществляем выход из Cisco-меню
 */
func (c *Connection) CiscoMenuAction(spawnOutput string, prompt *Prompt) error {
	buttonQ := regexp.MustCompile(`\sq\s.*(exit|quit|close)`)
	buttonE := regexp.MustCompile(`\se\s.*(exit|quit|close)`)
	buttonC := regexp.MustCompile(`\sc\s.*(exit|quit|close)`)

	switch {
	case buttonQ.MatchString(spawnOutput):
		return c.spawn.SendExitFromMenuCisco("q", prompt)
	case buttonE.MatchString(spawnOutput):
		return c.spawn.SendExitFromMenuCisco("e", prompt)
	case buttonC.MatchString(spawnOutput):
		return c.spawn.SendExitFromMenuCisco("c", prompt)
	}

	return errors.New(ports.ERROR_INTERNAL_CISCO_MENU_EXIT)
}

/*
 * Connection.PromptDefine
 *
 * Определяем тип устройства (его Prompt)
 */
func (c *Connection) PromptDefine() error {

	// Отправляем пустую команду для корректного отображения prompt строки
	output, sendError := c.spawn.SendString("", ports.SPAWN_TIMEOUT_SYSTEM, &PromptUniversal)
	if sendError != nil {
		return sendError
	}

	// Некоторые устройства некорректно отправляют свой Prompt и не улавливается
	// часть с переносом строки, хотя она по факту есть, для этого выполним проверку
	// наличая в начале символов переноса строки и если таких символов нет, то добавим
	if !strings.HasPrefix(output, "\r\n") {
		output = "\r\n" + output
	}

	logger.DEBUG(fmt.Sprintf("SPAWNER_PROMPT_DEF: Prompt verify using string: '%s'", output))

	// Определяем текущий prompt на основе возвращённого вывода
	prompt, promptError := NewPrompt(output)
	if promptError != nil {
		return promptError
	}

	logger.DEBUG(fmt.Sprintf("SPAWNER_PROMPT_DEF: Prompt changed to: '%s'", prompt.Name))

	// Устанавливаем захваченный prompt как текущий
	c.Prompt = prompt

	// На некоторых типах устройства перед тем как отдать управление пользователю
	// нужно предпринять действия по переходу в корректных режим управления
	// Например, на Cisco есть две ситуации, когда нужно предпринять различные действия:
	//  1) Если мы попали в пользовательский режим (откуда невозможно корректно выполнять
	// команды), необходимо перейти в привелигированный режим
	//  2) Если на оборудовании Cisco при запуске настроен вывод меню, то из этого меню
	// нужно выйти, что бы попасть в привелигированный режим
	switch c.Prompt.Name {

	// Если текущий Prompt - Cisco User, то переходим в Cisco Privilege
	case PromptCiscoUser.Name:
		{
			c.spawn.SendEnableCisco(&PromptCiscoPriv)
			c.PromptDefine()
			if c.Prompt.Name != PromptCiscoPriv.Name {
				return errors.New(ports.ERROR_INTERNAL_CISCO_ENABLE)
			}
		}

	// Если текущий Prompt - это Cisco Menu (всплывающее меню), настроенное на автозапуск
	// для линии VTY, то вводим команды "q", "c" или "e" для выхода из него
	case PromptCiscoMenu.Name:
		{
			c.CiscoMenuAction(c.connectOutput, &PromptCiscoPriv)
			c.PromptDefine()
			if c.Prompt.Name != PromptCiscoPriv.Name {
				return errors.New(ports.ERROR_INTERNAL_CISCO_MENU_EXIT)
			}
		}
	}

	return nil
}

/*
 * Connection.Close
 *
 * Закрываем сессию к удалённому устройству
 */
func (c *Connection) Close() {
	if c.spawn.Session != nil {
		c.spawn.Session.Close()
	}
}
