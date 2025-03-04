//go:generate easyjson -all main.go
package hw3

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/mailru/easyjson"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	// "log"
	//"github.com/mailru/easyjson"
)

//easyjson:json
type User struct {
	Browsers []string `json:"browsers"`
	Company  string   `json:"company"`
	Country  string   `json:"country"`
	Email    string   `json:"email"`
	Job      string   `json:"job"`
	Name     string   `json:"name"`
	Phone    string   `json:"phone"`
}

const filePath string = "./data/users.txt"

func SlowSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	r := regexp.MustCompile("@")
	seenBrowsers := make(map[string]struct{})
	uniqueBrowsers := 0
	var foundUsers strings.Builder

	scanner := bufio.NewScanner(file)
	i := 0

	for scanner.Scan() {
		var user User
		if err := easyjson.Unmarshal([]byte(scanner.Text()), &user); err != nil {
			continue
		}

		isAndroid, isMSIE := false, false

		for _, browser := range user.Browsers {
			if strings.Contains(browser, "Android") || strings.Contains(browser, "MSIE") {
				if _, exists := seenBrowsers[browser]; !exists {
					seenBrowsers[browser] = struct{}{}
					uniqueBrowsers++
				}

				if strings.Contains(browser, "Android") {
					isAndroid = true
				}
				if strings.Contains(browser, "MSIE") {
					isMSIE = true
				}
			}
		}

		if isAndroid && isMSIE {
			email := r.ReplaceAllString(user.Email, " [at] ")
			foundUsers.WriteString(fmt.Sprintf("[%d] %s <%s>\n", i, user.Name, email))
		}

		i++
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	fmt.Fprintln(out, "found users:\n"+foundUsers.String())
	fmt.Fprintln(out, "Total unique browsers", uniqueBrowsers)
}

func SlowSearch1(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	r := regexp.MustCompile("@")
	seenBrowsers := []string{}
	uniqueBrowsers := 0
	foundUsers := ""

	lines := strings.Split(string(fileContents), "\n")

	users := make([]map[string]interface{}, 0)
	for _, line := range lines {
		user := make(map[string]interface{})
		// fmt.Printf("%v %v\n", err, line)
		err := json.Unmarshal([]byte(line), &user)
		if err != nil {
			panic(err)
		}
		users = append(users, user)
	}

	for i, user := range users {

		isAndroid := false
		isMSIE := false

		browsers, ok := user["browsers"].([]interface{})
		if !ok {
			// log.Println("cant cast browsers")
			continue
		}

		for _, browserRaw := range browsers {
			browser, ok := browserRaw.(string)
			if !ok {
				// log.Println("cant cast browser to string")
				continue
			}
			if ok, err := regexp.MatchString("Android", browser); ok && err == nil {
				isAndroid = true
				notSeenBefore := true
				for _, item := range seenBrowsers {
					if item == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			}
		}

		for _, browserRaw := range browsers {
			browser, ok := browserRaw.(string)
			if !ok {
				// log.Println("cant cast browser to string")
				continue
			}
			if ok, err := regexp.MatchString("MSIE", browser); ok && err == nil {
				isMSIE = true
				notSeenBefore := true
				for _, item := range seenBrowsers {
					if item == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}

		// log.Println("Android and MSIE user:", user["name"], user["email"])
		email := r.ReplaceAllString(user["email"].(string), " [at] ")
		foundUsers += fmt.Sprintf("[%d] %s <%s>\n", i, user["name"], email)
	}

	fmt.Fprintln(out, "found users:\n"+foundUsers)
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}
