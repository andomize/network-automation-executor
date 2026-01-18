package logger

import (
	"log"
)

func DEBUG(message string) {
	if debugMode {
		log.Println("[" + eDEBUG.PRI + "] " + message)
	}
}

func INFO(message string) {
	log.Println("[" + eINFO.PRI + "] " + message)
}

func WARNING(message string) {
	log.Println("[" + eWARNING.PRI + "] " + message)
}

func ERROR(message string) {
	log.Println("[" + eERROR.PRI + "] " + message)
}

func ModuleEnableDebug() {
	debugMode = true
}

func ModuleDisableDebug() {
	debugMode = false
}

func ModuleIsDebug() bool {
	return debugMode
}

func Must(err error, message string) {
	if err != nil {
		if len(message) > 0 {
			ERROR(message)
		}
		log.Fatal(err)
	}
}
