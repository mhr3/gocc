package asm

import "regexp"

func getRegexpParams(re *regexp.Regexp, text string) map[string]string {
	match := re.FindStringSubmatch(text)
	res := map[string]string{}
	for i, name := range re.SubexpNames() {
		if name == "" {
			continue
		}
		res[name] = match[i]
	}

	return res
}
