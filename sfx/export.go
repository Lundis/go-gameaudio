package sfx

import (
	"strings"
)

// ExportConstants is used to export all currently loaded SFX,
// in a format that can be used to generate go constants.
func ExportConstants() map[string]string {
	export := make(map[string]string)
	for id := range loadedSfx {
		formattedConstant := ""
		capsNext := true
		for i := 0; i < len(id); i++ {
			c := id[i]
			if c == '-' || c == '_' || c == '.' || c == ' ' {
				capsNext = true
				i++
			} else {
				if capsNext {
					c = strings.ToUpper(string(c))[0]
					capsNext = false
				}
				formattedConstant += string(c)
			}
		}
		export[formattedConstant] = string(id)
	}
	return export
}
