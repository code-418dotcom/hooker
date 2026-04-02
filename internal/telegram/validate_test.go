package telegram

import "testing"

func TestIsValidName(t *testing.T) {
	valid := []string{
		"nginx",
		"my-container",
		"my_container",
		"my.container",
		"web01",
		"a",
		"Container-Name_v2.1",
	}
	for _, name := range valid {
		if !isValidName(name) {
			t.Errorf("expected %q to be valid", name)
		}
	}

	invalid := []string{
		"",
		"-starts-with-dash",
		".starts-with-dot",
		"_starts-with-underscore",
		"has spaces",
		"has/slash",
		"../traversal",
		"name\nwith\nnewlines",
		"has;semicolon",
		string(make([]byte, 129)), // too long
	}
	for _, name := range invalid {
		if isValidName(name) {
			t.Errorf("expected %q to be invalid", name)
		}
	}
}
