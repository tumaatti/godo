package main

import (
	"testing"
)

func TestFormatCheckMarkTrue(t *testing.T) {
	want := "[X]"
	done := formatCheckMark(true)

	if want != done {
		t.Fatalf(`formatCheckMark(true) = %q, want match for %#q, nil`, done, want)
	}
}

func TestFormatCheckMarkFalse(t *testing.T) {
	want := "[ ]"
	done := formatCheckMark(false)

	if want != done {
		t.Fatalf(`formatCheckMark(false) = %q, want match for %#q`, done, want)
	}
}

func TestUnFormatCheckMarkTrue(t *testing.T) {
	want := true
	done := unFormatCheckMark("[X]")
	if want != done {
		t.Fatalf(`unFormatCheckMark(true) = %t, want match for %t`, done, want)
	}
}

func TestUnFormatCheckMarkFalse(t *testing.T) {
	want := false
	done := unFormatCheckMark("[ ]")
	if want != done {
		t.Fatalf(`unFormatCheckMark(false) = %t, want match for %t`, done, want)
	}
}

func TestParseFile(t *testing.T) {
	testContent := []byte("DONE:[ ];TAGS:penispenis\nsome text")
	tmpFile := File{content: testContent}

	wantedContent := "some text"
	wantedTags := "penispenis"
	wantedDone := false
	content, tags, done := tmpFile.parseFile()

	if wantedContent != content {
		t.Fatalf(`content = %q, wantedContent %q`, content, wantedContent)
	}

	if wantedTags != tags {
		t.Fatalf(`content = %q, wantedContent %q`, tags, wantedTags)
	}

	if wantedDone != done {
		t.Fatalf(`content = %t, wantedContent %t`, done, wantedDone)
	}
}

func TestFilterArgsList(t *testing.T) {
	arg := "--list"
	testArgs := []string{arg}
	commands := filterArguments(testArgs)
	args, ok := commands[arg]

	if !ok || len(args) != 0 {
		t.Fatalf(`ok = %t, len(args) = %d expected: ok = true len(args) = 0`, ok, len(args))
	}
}

func TestFilterArgsListTags(t *testing.T) {
	testArgs := []string{"--list", "--tag", "penis"}
	commands := filterArguments(testArgs)

	listArgs, listOk := commands["--list"]

	if !listOk || len(listArgs) != 0 {
		t.Fatalf(`ok = %t, len(args) = %d expected: ok = true len(args) = 0`, listOk, len(listArgs))
	}

	tagsArgs, tagsOk := commands["--tag"]

	wanted := "penis"

	if !tagsOk || tagsArgs[0] != wanted {
		t.Fatalf(`ok = %t, tagsArgs = %q expected: ok = true tagsArgs[0] = penis`, tagsOk, tagsArgs[0])
	}

}

func TestGetKeyArgs(t *testing.T) {
}
