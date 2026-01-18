package logger

type Severity struct {
	PRI  string
	CODE int
}

var eDEBUG = Severity{PRI: "DEBUG", CODE: 0}         // Severity 0
var eINFO = Severity{PRI: "INFO", CODE: 1}           // Severity 1
var eNOTICE = Severity{PRI: "NOTICE", CODE: 2}       // Severity 2
var eWARNING = Severity{PRI: "WARNING", CODE: 3}     // Severity 3
var eERROR = Severity{PRI: "ERROR", CODE: 4}         // Severity 4
var eCRITICAL = Severity{PRI: "CRITICAL", CODE: 5}   // Severity 5
var eALERT = Severity{PRI: "ALERT", CODE: 6}         // Severity 6
var eEMERGENCY = Severity{PRI: "EMERGENCY", CODE: 7} // Severity 7

var debugMode bool = false
