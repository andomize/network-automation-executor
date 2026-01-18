package spawner

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/andomize/network-automation-executor/internal/adapters/logger"
	"github.com/andomize/network-automation-executor/internal/core/ports"
)

type Prompt struct {
	// Vendor
	Vendor string

	// Name of prompt
	Name string

	// Regular expression that define prompt
	RegExp *regexp.Regexp

	// Regular expression that define error messages
	Errors []string
}

var (
	// Universal prompt using by default and include prompt that
	// describe any devices and general errors by connection establishing
	PromptUniversal = Prompt{
		Name:   "universal",
		Vendor: "unknown",
		RegExp: regexp.MustCompile(`[^>#:]([>#:\]]\s?)$`),
		Errors: []string{
			`([Cc]onnection\sclosed)`,
			`([Aa]uthentication\sfailed)`,
			`([Cc]onnection\srefused)`,
			`([Cc]onnection\stimed\sout)`,
			`([Pp]ermission\sdenied)`,
			`([Tt]he\sremote\ssystem\srefused\sthe\sconnection)`,
			`([Uu]nable\sto\snegotiate\swith)`,
		},
	}

	PromptCiscoUser = Prompt{
		Name:   "cisco-user",
		Vendor: "cisco",
		RegExp: regexp.MustCompile(`\r\n\r?[^<\s]+>`),
		Errors: []string{
			`(\n\r?[Tt]ranslating.*domain server)`,        // Translating "a"...domain server...
			`(\n\r?%\s[Bb]ad\sIP\saddress)`,               // % Bad IP address
			`(\n\r?%\s[Uu]nknown\scommand)`,               // % Unknown command
			`(\n\r?%\s[Ii]ncomplete\scommand)`,            // % Incomplete command
			`(\n\r?%\s[Aa]mbiguous\scommand)`,             // % Ambiguous command
			`(\n\r?%\s[Ii]nvalid\sinput)`,                 // % Invalid input detected at '^' marker.
			`(\n\r?%\s[Ii]nvalid\s[Cc]ommand)`,            // % Invalid Command at '^' marker (FXOS)
			`(\n\r?%\s[Aa]ccess\sdenied)`,                 // % Access denied
			`(\n\r?%\s[Ee]rror\sin\sauthentication)`,      // % Error in authentication
			`(\n\r?[Uu]nrecognized\shost)`,                // Unrecognized host
			`(\n\r?[Cc]ommand\sauthorization\sfailed)`,    // Command authorization failed
			`(\n\r?[Cc]ommand\srejected:)`,                // Command rejected: Bad VLAN list...
			`(\n\r?ERROR:\s%\s[Ii]nvalid\s[Ii]nput)`,      // ERROR: % Invalid input
			`(\n\r?ERROR:\s%\s[Ii]nvalid\s[Hh]ostname)`,   // ERROR: % Invalid Hostname
			`(\n\r?ERROR:\s%\s[Ii]ncomplete\s[Cc]ommand)`, // ERROR: % Incomplete command
			`(\n\r?%[Ee]rror\sparsing\sfilename)`,         // %Error parsing filename
			`(\n\r?%[Ee]rror\sopening)`,                   // %Error opening
			// Inactive timeout reached, logging out.
			// The idle timeout is soon to expire on this line
			// timed out waiting for input: auto-logout
		},
	}

	PromptCiscoPriv = Prompt{
		Name:   "cisco-priv",
		Vendor: PromptCiscoUser.Vendor,
		RegExp: regexp.MustCompile(`\r?\n\r?[^#\s]+#`),
		Errors: PromptCiscoUser.Errors,
	}

	PromptCiscoConf = Prompt{
		Name:   "cisco-conf",
		Vendor: PromptCiscoUser.Vendor,
		RegExp: regexp.MustCompile(`\r?\n\r?[^#\s]+\(conf[^#\s]+?\)#`),
		Errors: PromptCiscoUser.Errors,
	}

	PromptCiscoMenu = Prompt{
		Name:   "cisco-menu",
		Vendor: PromptCiscoUser.Vendor,
		RegExp: regexp.MustCompile(`.*([Ss]elect\s[Aa]ction|[Yy]our\s[Ss]election).*:`),
		Errors: PromptCiscoUser.Errors,
	}

	PromptHuaweiUser = Prompt{
		Name:   "huawei-user",
		Vendor: "huawei",
		RegExp: regexp.MustCompile(`\r?\n\r?(.+)?<.+>`),
		Errors: []string{
			`(\r\n\r?[Ee]rror:\s)`,
			// The server has disconnected with an error.
			// Info: The max number of VTY users is 5, and the number of current VTY users on line is 0.
		},
	}

	PromptHuaweiSys = Prompt{
		Name:   "huawei-sys",
		Vendor: PromptHuaweiUser.Vendor,
		RegExp: regexp.MustCompile(`\r?\n\r?(.+)?\[.+\]`),
		Errors: PromptHuaweiUser.Errors,
	}

	PromptF5Bash = Prompt{
		Name:   "f5-bash",
		Vendor: "f5",
		// [<login user>@<device hostname>:<device state>:<device group sync status>]
		RegExp: regexp.MustCompile(`\[[a-zA-Z0-9\-\_]+?@[a-zA-Z0-9\-\_]+?\:[a-zA-Z\s]+?\:[a-zA-Z\s]+?\]`),
		Errors: []string{
			`(\-bash:\s.*:\scommand\snot\sfound)`,
		},
	}

	PromptF5TMSH = Prompt{
		Name:   "f5-tmos",
		Vendor: PromptF5Bash.Vendor,
		// <login user>@(<device hostname>)(cfg-sync <device group sync status>)(<device state>)
		RegExp: regexp.MustCompile(`[a-zA-Z0-9\-\_]+?\@\([a-zA-Z0-9\-\_]+?\)\([a-zA-Z0-9\-\_\s]+?\)\([a-zA-Z0-9\-\_\s]+?\)\([a-zA-Z0-9\-\_\s\/]+?\)\(tmos\)`),
		Errors: []string{
			`([Ss]yntax\s[Ee]rror:)`,
			`([Uu]nexpected\s[Ee]rror:)`,
			`([Uu]se\s\"quit\"\sto\send\sthe\scurrent\ssession)`,
		},
	}

	PromptRadwareAlteon = Prompt{
		Name:   "radware-alteon",
		Vendor: "radware",
		// >> Main#
		// >> Operations#
		// >> Border Gateway Protocol Operations#
		RegExp: regexp.MustCompile(`\r?\n\r?>>\s[^#]+#`),

		// Error: unknown command "prpint"
		// Error: no parameter(s) expected
		Errors: []string{
			`(\r\n\r?[Ee]rror:\s)`,
		},
	}
)

func (p *Prompt) GetUniversalExp() *regexp.Regexp {
	// Get universal expression
	return PromptUniversal.RegExp
}

func (p *Prompt) GetRegExp() *regexp.Regexp {
	// Get expression to define device prompt
	if p != nil && p.RegExp != nil {
		return p.RegExp
	} else {
		return PromptUniversal.RegExp
	}
}

func (p *Prompt) GetErrors() *regexp.Regexp {
	// Create expression for errors verify using current prompt
	if p != nil && len(p.Name) > 0 {
		return regexp.MustCompile(strings.Join(p.Errors, "|"))
	} else {
		return regexp.MustCompile(strings.Join(PromptUniversal.Errors, "|"))
	}
}

func NewPrompt(output string) (*Prompt, error) {

	// Using console output to choose correct device prompt
	switch {
	case PromptCiscoConf.RegExp.MatchString(output):
		return &PromptCiscoConf, nil
	case PromptCiscoUser.RegExp.MatchString(output):
		return &PromptCiscoUser, nil
	case PromptCiscoPriv.RegExp.MatchString(output):
		return &PromptCiscoPriv, nil
	case PromptCiscoMenu.RegExp.MatchString(output):
		return &PromptCiscoMenu, nil
	case PromptHuaweiUser.RegExp.MatchString(output):
		return &PromptHuaweiUser, nil
	case PromptHuaweiSys.RegExp.MatchString(output):
		return &PromptHuaweiSys, nil
	case PromptF5Bash.RegExp.MatchString(output):
		return &PromptF5Bash, nil
	case PromptF5TMSH.RegExp.MatchString(output):
		return &PromptF5TMSH, nil
	case PromptRadwareAlteon.RegExp.MatchString(output):
		return &PromptRadwareAlteon, nil
	}

	logger.DEBUG("PROMPT_NEW: Cannot define prompt: '" + output + "'")
	logger.DEBUG("PROMPT_NEW: Cannot define prompt: '" + fmt.Sprint([]byte(output)) + "'")
	logger.DEBUG("PROMPT_NEW: Cannot define prompt: '" + ByteDebugInterpreter(output) + "'")

	return nil, errors.New(ports.ERROR_PROMPT_DEFINE)
}

func ByteDebugInterpreter(str string) string {

	var pseudoOutput string

	for _, value := range []byte(str) {

		var byteCode string

		switch value {
		case byte(10):
			byteCode += "\\n"
		case byte(13):
			byteCode += "\\r"
		case byte(32):
			byteCode += "\\s"
		default:
			byteCode += string(value)
		}

		pseudoOutput += byteCode
	}

	return pseudoOutput
}
