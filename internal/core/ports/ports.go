package ports

// Версия программы. Только в формате 0.0.0
const VERSION = "1.1.4"

// Системное время по умолчанию для подключения к устройству - 30 секунд
const SPAWN_TIMEOUT_SYSTEM = 20

// Возможные состояния задания
const PIPE_STATUS_SUCCESS = "success"
const PIPE_STATUS_FAIL = "fail"
const PIPE_STATUS_SKIPPED = "skipped"

// Ошибки при установлении сессии

const ERROR_CONN_CLOSED = "connection-closed"
const ERROR_CONN_AUTH_FAIL = "connection-auth-fail"
const ERROR_CONN_REFUSED = "connection-refused"
const ERROR_CONN_TIMEOUT = "connection-timeout"
const ERROR_CONN_DENIED = "connection-denied"
const ERROR_CONN_UNABLE_TO_NEGOTIATE = "connection-unable-to-negotiate"
const ERROR_CONN_NO_AVAILABLE_METHOD = "connection-no-available-method"

// Типовые ошибки при отправке команд

const ERROR_SEND_COMMAND = "spawner-command-send-error"
const ERROR_PROMPT_TIMEOUT = "spawner-prompt-capture-timeout"
const ERROR_PROMPT_CHANGED = "spawner-prompt-has-been-changed"
const ERROR_PROMPT_DEFINE = "spawner-prompt-was-not-defined"

// Ошибки расширенного функционала

const ERROR_REGEX_VAR_NOT_EXIST = "spawner-regex-variable-not-exist"
const ERROR_REGEX_GROUP_NE = "spawner-regex-group-val-count-not-equal"
const ERROR_WHEN_CONDITION_DOUBLE_BASED = "spawner-when-condition-double-based"

// Ошибки форматирования файла задания

const ERROR_SYNTAX_NO_HOST = "syntax-host-is-not-set"
const ERROR_SYNTAX_NO_TASKS = "syntax-no-tasks"

// Внутренние ошибки

const ERROR_INTERNAL_EXEC = "internal-error-spawn-exec-command-error"
const ERROR_INTERNAL_BUFFER = "internal-error-spawn-buffer-is-empty"
const ERROR_INTERNAL_SSHHELLO = "internal-error-sshhello-yes-send"
const ERROR_INTERNAL_CISCO_ENABLE = "internal-error-cisco-enable"
const ERROR_INTERNAL_CISCO_MENU_EXIT = "internal-error-cisco-menu-exit"
