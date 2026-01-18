package domains

type TaskPattern struct {
	Host          string            `json:"host,omitempty"`
	Status        string            `json:"status,omitempty"`
	Vendor        string            `json:"vendor,omitempty"`
	Error         string            `json:"error,omitempty"`
	CreatinGtime  string            `json:"creatingtime,omitempty"`
	ExecutingTime string            `json:"executingtime,omitempty"`
	Tasks         *[]Task           `json:"tasks"`
	Settings      *Setting          `json:"settings,omitempty"`
	Variables     map[string]string `json:"variables,omitempty"`
	Autotests     *[]When           `json:"autotests,omitempty"`
}

type Setting struct {
	Timeout int `json:"timeout,string,omitempty"`
}

type Task struct {
	Command string  `json:"command,omitempty"`
	Status  string  `json:"status,omitempty"`
	Name    string  `json:"name,omitempty"`
	Params  Param   `json:"params"`
	Tasks   *[]Task `json:"tasks,omitempty"`
	When    *[]When `json:"when,omitempty"`
}

type Param struct {
	Timeout              int    `json:"timeout,string,omitempty"`
	OutputFile           string `json:"outputFile,omitempty"`
	OnErrorContinue      bool   `json:"onErrorContinue,string,omitempty"`
	PromptChangeAllowed  bool   `json:"promptChangeAllowed,string,omitempty"`
	CommandRepeatAllowed bool   `json:"commandRepeatAllowed,string,omitempty"`
	Filter               string `json:"filter,omitempty"`
	FilterExclude        string `json:"filterExclude,omitempty"`
}

type When struct {
	// Task-Name-Based Conditional Statements
	Name                  string `json:"name,omitempty"`
	IfStatus              string `json:"ifStatus,omitempty"`
	IfOutputContains      string `json:"ifOutputContains,omitempty"`
	IfOutputNotContains   string `json:"ifOutputNotContains,omitempty"`
	IfOutputContainsRe    string `json:"ifOutputContainsRe,omitempty"`
	IfOutputNotContainsRe string `json:"ifOutputNotContainsRe,omitempty"`

	// Variable-Based Conditional Statements
	Variable   string `json:"variable,omitempty"`
	IfValue    string `json:"ifValue,omitempty"`
	IfValueNot string `json:"ifValueNot,omitempty"`

	// Actions
	OnMove string `json:"onMove,omitempty"`
	OnExit bool   `json:"onExit,string,omitempty"`
}
