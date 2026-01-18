package spawner

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/andomize/network-automation-executor/internal/adapters/logger"
	"github.com/andomize/network-automation-executor/internal/core/ports"
	expect "github.com/google/goexpect"
	"google.golang.org/grpc/codes"
)

type Spawn struct {

	// Экземпляр библиотеки https://github.com/google/goexpect
	// Предназначен для взаимодействия с удалённым устрйоством, а именно:
	// - Открывает удалённую сессию благодаря запуску ssh/telnet
	// - Отправляет команды на удалённое устройство
	// - Считывает вывод с удалённого устройства и помещает в буфер
	// Поддерживается только в среде Linux / Docker (На Windows не работает)
	Session *expect.GExpect

	Username string
	Password string
}

func NewSpawn(username, password, bashCommand string) (*Spawn, string, error) {

	// Создаём экземпляр Spawn сессии
	spawn := Spawn{
		Username: username,
		Password: password,
	}

	// Открываем Spawn сессию
	output, openError := spawn.Open(bashCommand)
	if openError != nil {
		return nil, output, openError
	}

	return &spawn, output, nil
}

/*
 * Spawn.Open
 *
 * Authentication on remote device using specific command
 */
func (s *Spawn) Open(bashCommand string) (string, error) {

	// Spawn starts a new process and collects the output. The error channel
	// returns the result of the command Spawned when it finishes.
	server, _, spawnError := expect.Spawn(bashCommand, -1)
	if spawnError != nil {
		logger.DEBUG("SPAWN_OPEN: Cannot create spawn session by error: " + spawnError.Error())
		return "", errors.New(ports.ERROR_INTERNAL_EXEC)
	}

	logger.DEBUG("SPAWN_OPEN: Spawn command: '" + bashCommand + "'")
	// ExpectBatch takes an array of BatchEntry and executes them in order
	// filling in the BatchRes array for any Expect command executed.
	resources, connectionError := server.ExpectBatch([]expect.Batcher{
		&expect.BCas{C: []expect.Caser{

			// # "Are you sure you want to continue connecting (yes/no)?", send: "yes"
			&expect.Case{R: regexp.MustCompile(`yes.no`), S: "yes\n", T: expect.Continue(
				expect.NewStatus(codes.Canceled, ports.ERROR_INTERNAL_SSHHELLO)), Rt: 1},

			// # Password required message: "Username:", send username
			&expect.Case{R: regexp.MustCompile(`[Uu]sername:`), S: s.Username + "\n",
				T: expect.Continue(expect.NewStatus(codes.Canceled, ports.ERROR_CONN_AUTH_FAIL)), Rt: 1},

			// # Password required message: "Password:", send password
			&expect.Case{R: regexp.MustCompile(`[Pp]assword:`), S: s.Password + "\n",
				T: expect.Continue(expect.NewStatus(codes.Canceled, ports.ERROR_CONN_AUTH_FAIL)), Rt: 1},

			// # Check connection errors: "Connection closed", "Connection refused", etc...
			&expect.Case{R: regexp.MustCompile(PromptUniversal.Errors[0]), T: expect.Fail(
				expect.NewStatus(codes.Canceled, ports.ERROR_CONN_CLOSED))},
			&expect.Case{R: regexp.MustCompile(PromptUniversal.Errors[1]), T: expect.Fail(
				expect.NewStatus(codes.Canceled, ports.ERROR_CONN_AUTH_FAIL))},
			&expect.Case{R: regexp.MustCompile(PromptUniversal.Errors[2]), T: expect.Fail(
				expect.NewStatus(codes.Canceled, ports.ERROR_CONN_REFUSED))},
			&expect.Case{R: regexp.MustCompile(PromptUniversal.Errors[3]), T: expect.Fail(
				expect.NewStatus(codes.Canceled, ports.ERROR_CONN_TIMEOUT))},
			&expect.Case{R: regexp.MustCompile(PromptUniversal.Errors[4]), T: expect.Fail(
				expect.NewStatus(codes.Canceled, ports.ERROR_CONN_DENIED))},
			&expect.Case{R: regexp.MustCompile(PromptUniversal.Errors[5]), T: expect.Fail(
				expect.NewStatus(codes.Canceled, ports.ERROR_CONN_REFUSED))},
			&expect.Case{R: regexp.MustCompile(PromptUniversal.Errors[6]), T: expect.Fail(
				expect.NewStatus(codes.Canceled, ports.ERROR_CONN_UNABLE_TO_NEGOTIATE))},

			// # Check connection using universal prompt output
			&expect.Case{R: PromptUniversal.RegExp, T: expect.OK()},
		}},
	}, time.Duration(ports.SPAWN_TIMEOUT_SYSTEM)*time.Second)

	if resources == nil || len(resources) <= 0 {
		return "", errors.New(ports.ERROR_INTERNAL_BUFFER)
	}

	logger.DEBUG("SPAWN_OPEN: RAW:" + fmt.Sprint(resources))

	if connectionError != nil {
		if strings.Contains(connectionError.Error(), "expect: Process not running") {
			// Если ошибка содержит в себе фразу "expect: Process not running"
			// то процесс не был запущен из-за длительного ожидания
			return resources[0].Output, errors.New(ports.ERROR_CONN_TIMEOUT)
		}
		if strings.Contains(connectionError.Error(), "expect: timer expired") {
			// Authentication failed by reason - timer expired
			// Convert error code by proprietary format
			return resources[0].Output, errors.New(ports.ERROR_CONN_TIMEOUT)
		}
		return resources[0].Output, connectionError
	}

	// Подключение успешно
	s.Session = server
	return resources[0].Output, nil
}

/*
 * Spawn.SendString
 *
 * Send string command to remote device and read output using universal prompt
 * Result will be returned as string with removed \r\n tags around output
 */
func (s *Spawn) SendString(command string, timeout int, prompt *Prompt) (string, error) {

	logger.DEBUG("SPAWNER_SEND_STR: Command: '" + command + "'")
	logger.DEBUG("SPAWNER_SEND_STR: Prompt Name: '" + prompt.Name + "'")
	logger.DEBUG("SPAWNER_SEND_STR: PromptRegExp: '" + prompt.GetRegExp().String() + "'")

	// Send command to remote device and read output
	resources, connectionError := s.Session.ExpectBatch([]expect.Batcher{
		&expect.BSnd{S: command + "\n"},
		&expect.BCas{C: []expect.Caser{
			// Errors verify
			&expect.Case{R: prompt.GetErrors(), T: expect.Fail(
				expect.NewStatus(codes.Canceled, ports.ERROR_SEND_COMMAND))},
			// Prompt OK
			&expect.Case{R: prompt.GetRegExp(), T: expect.OK()},
		}},
	}, time.Duration(timeout)*time.Second)

	if resources == nil || len(resources) <= 0 {
		return "", errors.New(ports.ERROR_INTERNAL_BUFFER)
	}

	logger.DEBUG("SPAWNER_SEND_STR: RAW:" + fmt.Sprint(resources))

	if connectionError != nil {
		if strings.Contains(connectionError.Error(), "expect: timer expired") {
			// Authentication failed by reason - timer expired
			// Convert error code by proprietary format
			return resources[0].Output, errors.New(ports.ERROR_PROMPT_TIMEOUT)
		}
	}

	return resources[0].Output, connectionError
}

/*
 * Spawn.SendEnableCisco
 *
 * Send command `enable` to remote device
 */
func (s *Spawn) SendEnableCisco(prompt *Prompt) error {

	logger.DEBUG("SPAWNER_SEND_ENABLE: Command: 'enable'")

	// Send "enable" command and password to remote device
	res, err := s.Session.ExpectBatch([]expect.Batcher{
		&expect.BSnd{S: "enable\n"},
		&expect.BCas{C: []expect.Caser{
			// # Password required message: "Password:", send: password
			&expect.Case{R: regexp.MustCompile(`[Pp]assword:`), S: s.Password + "\n",
				T: expect.Continue(expect.NewStatus(codes.Canceled, ports.ERROR_INTERNAL_CISCO_ENABLE)), Rt: 1},
			// Ожидаем новый тип Prompt - Cisco Privilege
			&expect.Case{R: prompt.RegExp, T: expect.OK()},
		}},
	}, time.Duration(ports.SPAWN_TIMEOUT_SYSTEM)*time.Second)

	if res == nil {
		return errors.New(ports.ERROR_INTERNAL_BUFFER)
	}

	logger.DEBUG("SPAWNER_SEND_ENABLE: RAW:" + fmt.Sprint(res))

	return err
}

/*
 * Spawn.SendExitFromMenuCisco
 *
 * Send exit command to remote device
 */
func (s *Spawn) SendExitFromMenuCisco(command string, prompt *Prompt) error {

	logger.DEBUG("SPAWNER_SEND_MENU_EXIT: Command: '" + command + "'")

	// Send "e" command to exit from cisco console menu
	res, err := s.Session.ExpectBatch([]expect.Batcher{
		&expect.BSnd{S: command + "\n"},
		&expect.BCas{C: []expect.Caser{
			// Ожидаем новый тип Prompt - Cisco Privilege
			&expect.Case{R: prompt.RegExp, T: expect.OK()},
		}},
	}, time.Duration(ports.SPAWN_TIMEOUT_SYSTEM)*time.Second)

	if res == nil {
		return errors.New(ports.ERROR_INTERNAL_BUFFER)
	}

	logger.DEBUG("SPAWNER_SEND_MENU_EXIT: RAW:" + fmt.Sprint(res))

	return err
}
