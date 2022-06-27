//A Packager Server Written in Go
//Author: Xinchi Qi
package main

import (
	"bufio"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
)

const (
	host = "localhost"
	port = "8080"
)

/* var packageList = struct {
	sync.RWMutex
	m map[string]map[string]string
}{m: make(map[string]map[string]string)} */

var packageList = make(map[string]map[string]string)
var mutex = &sync.RWMutex{}

func checkFormat(message string) bool {
	splitString := func(c rune) bool {
		return c == '|'
	}
	splitDependencies := func(c rune) bool {
		return c == ','
	}
	splitPackageName := func(c rune) bool {
		return c == '-'
	}
	var isStringAlphabetic = regexp.MustCompile(`^[a-zA-Z0-9_+-]*$`).MatchString

	fields := strings.FieldsFunc(message, splitString)
	fieldLength := len(fields)
	fmt.Println(message)
	//fmt.Println(fieldLength)
	if fieldLength == 2 || fieldLength == 3 {
		if fields[0] == "INDEX" || fields[0] == "REMOVE" || fields[0] == "QUERY" {
			payload := fields[1]
			if isStringAlphabetic(payload) {
				fmt.Println("AlphabeticPassed")
				packageFields := strings.FieldsFunc(payload, splitPackageName)
				for _, packageField := range packageFields {
					if strings.Contains(packageField, "+") {
						if strings.Index(packageField, "+") < len(packageField)-3 {
							return false
						}
					}
				}

				if len(fields) == 2 {
					return true
				} else if len(fields) == 3 {
					dependencies := strings.FieldsFunc(fields[2], splitDependencies)
					for i := 1; i < len(dependencies)-1; i++ {
						if strings.Contains(dependencies[i], " ") {
							fmt.Println("dependency contains space")
							return false
						}
					}
					return true
				}
			}
		}
	}
	fmt.Println("field len failed")
	return false
}

//Check if dependencies are indexed
func dependenciesCheck(message []string) bool {
	mutex.RLock()
	for _, v := range message {
		if _, ok := packageList[v]; ok {
			mutex.RUnlock()
			return true
		}
	}
	mutex.RUnlock()
	return false
}

func removalDependenciesCheck(message string) bool {
	mutex.RLock()
	for _, value := range packageList {
		if len(value) != 0 {
			for _, dependency := range value {
				if dependency == message {
					mutex.RUnlock()
					return true
				}
			}
		}
	}
	mutex.RUnlock()
	return false
}

func handleConnection(c net.Conn) {
	fmt.Printf("Serving %s\n", c.RemoteAddr().String())
	for {
		//Read from TCP
		message, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		}

		//Remove whitespaces and split strings
		temp := strings.TrimSpace(string(message))
		//fmt.Println(temp)
		fmt.Println(checkFormat(temp))
		//Command logic
		if checkFormat(temp) {
			splitString := func(c rune) bool {
				return c == '|'
			}
			splitDependencies := func(c rune) bool {
				return c == ','
			}
			fields := strings.FieldsFunc(temp, splitString)
			messageLength := len(fields)
			command := fields[0]
			packageName := fields[1]
			if messageLength == 2 || messageLength == 3 {
				fmt.Println(messageLength)
				if command == "INDEX" {
					if messageLength == 2 {
						mutex.RLock()
						_, present := packageList[packageName]
						mutex.RUnlock()
						if present {
							mutex.Lock()
							delete(packageList, packageName)
							mutex.Unlock()
						}
						mutex.Lock()
						packageList[packageName] = map[string]string{}
						mutex.Unlock()
						fmt.Println("Indexed OK")
						c.Write([]byte("OK\n"))
					} else if messageLength == 3 {
						dependencies := strings.FieldsFunc(fields[2], splitDependencies)
						fmt.Println(dependenciesCheck(dependencies))
						if dependenciesCheck(dependencies) {
							mutex.RLock()
							_, present := packageList[packageName]
							mutex.RUnlock()
							if present {
								mutex.Lock()
								delete(packageList, packageName)
								packageList[packageName] = map[string]string{}
								for i := 1; i < len(dependencies)-1; i++ {
									packageList[packageName][dependencies[i]] = dependencies[i]
								}
								mutex.Unlock()
								c.Write([]byte("OK\n"))
							} else {
								mutex.Lock()
								packageList[packageName] = map[string]string{}
								for i := 1; i < len(dependencies)-1; i++ {
									packageList[packageName][dependencies[i]] = dependencies[i]
								}
								mutex.Unlock()
								c.Write([]byte("OK\n"))
							}
						} else {
							fmt.Println("Indexed FAIL")
							c.Write([]byte("FAIL\n"))
						}
					}
				}
				if command == "REMOVE" {
					//c.Write([]byte("OK\n"))
					mutex.RLock()
					_, present := packageList[packageName]
					mutex.RUnlock()
					if present {
						if removalDependenciesCheck(packageName) {
							c.Write([]byte("FAIL\n"))
						} else {
							mutex.Lock()
							delete(packageList, packageName)
							mutex.Unlock()
							c.Write([]byte("OK\n"))
						}
					} else {
						//c.Write([]byte("OK\n"))
						_, err := c.Write([]byte("OK\n"))
						if err != nil {
							fmt.Println(err)
							return
						}
					}
				}
				if command == "QUERY" {
					if _, ok := packageList[packageName]; ok {
						c.Write([]byte("OK\n"))
					} else {
						c.Write([]byte("FAIL\n"))
					}
				}
			}
		} else {
			fmt.Println("Sent ERROR")
			c.Write([]byte("ERROR\n"))
			//c.Close()
		}

		//Initialize TCP writer
		//writer := bufio.NewWriter(c)

		//c.Write([]byte(string(temp)))

	}
	c.Close() // No need, the client Timeout automatically.
}

func main() {
	l, err := net.Listen("tcp", host+":"+port)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Accept connection on port 8080...")
	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		go handleConnection(c)
	}

}
