package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
)

//easyjson:json
type User struct {
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Browsers []string `json:"browsers"`
}

var pool = sync.Pool{
	// New creates an object when the pool has nothing available to return.
	// New must return an interface{} to make it flexible. You have to cast
	// your type after getting it.
	New: func() interface{} {
		// Pools often contain things like *bytes.Buffer, which are
		// temporary and re-usable.
		return &bytes.Buffer{}
	},
}

// вам надо написать более быструю оптимальную этой функции

func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)

	if err != nil {
		panic(err)
	}
	defer file.Close()
	var foundUsers bytes.Buffer
	scanner := bufio.NewScanner(file)

	r := regexp.MustCompile("@")
	seenBrowsers := []string{}
	uniqueBrowsers := 0
	var user User

	for i := 0; scanner.Scan(); i++ {

		isAndroid := false
		isMSIE := false

		user.UnmarshalJSON(scanner.Bytes())
		if err != nil {
			panic(err)
		}

		browsers := user.Browsers

		if browsers == nil {
			continue
		}

		for _, browser := range browsers {
			if strings.Contains(browser, "Android") {
				isAndroid = true
				notSeenBefore := true
				for _, item := range seenBrowsers {
					if item == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			}
		}

		for _, browser := range browsers {
			if strings.Contains(browser, "MSIE") {
				isMSIE = true
				notSeenBefore := true
				for _, item := range seenBrowsers {
					if item == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}
		foundUsers.Write([]byte(fmt.Sprintf("[%d] %s <%s>\n", i, user.Name, r.ReplaceAllString(user.Email, " [at] "))))
	}

	fmt.Fprintln(out, "found users:\n"+foundUsers.String())
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}
