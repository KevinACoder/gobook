// Copyright © 2011-12 Qtrac Ltd.
// 
// This program or package and any associated files are licensed under the
// Apache License, Version 2.0 (the "License"); you may not use these files
// except in compliance with the License. You can get a copy of the License
// at: http://www.apache.org/licenses/LICENSE-2.0.
// 
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
    "bufio"
    "fmt"
    "io"
    "log"
    "os"
    "path/filepath"
    "regexp"
    "runtime"
    "safemap"
)

var workers = runtime.NumCPU() //assume there are four workers

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU()) // Use all the machine's cores
    if len(os.Args) != 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
        fmt.Printf("usage: %s <file.log>\n", filepath.Base(os.Args[0]))
        os.Exit(1)
    }
	//create two channel to organize the processing
    lines := make(chan string, workers*4) //send each line from log file to worker routine, assign line channel a small buffer to reduce the likelihood that worker routines being blocked
    done := make(chan struct{}, workers) //keep track of when all work has been done
    pageMap := safemap.New()
    go readLines(os.Args[1], lines) //#2, read lines from file
    processLines(done, pageMap, lines)
    waitUntil(done) //wait for all worker routines finished
    showResults(pageMap)
}

func readLines(filename string, lines chan<- string) {
    file, err := os.Open(filename)
    if err != nil {
        log.Fatal("failed to open the file:", err)
    }
    defer file.Close()
    reader := bufio.NewReader(file)
    for {
        line, err := reader.ReadString('\n')
        if line != "" {
			//each line is sent to lines channel, this step would be 
			//  blocked if the channel buffer is full, util another 
			//  routine receives one and free buffer
            lines <- line
        }
        if err != nil {
            if err != io.EOF {
                log.Println("failed to finish reading the file:", err)
            }
            break
        }
    }
    close(lines) //once all lines has been read, close the lines channel, 
	//this tell the receciver that there is no more data to receive
}

func processLines(done chan<- struct{}, pageMap safemap.SafeMap,
    lines <-chan string) {
    getRx := regexp.MustCompile(`GET[ \t]+([^ \t\n]+[.]html?)`)
	//the updater function
    incrementer := func(value interface{}, found bool) interface{} {
        if found {
			//use type assertion before increment
            return value.(int) + 1
        }
        return 1
    }
	//# 3, 4, 5, 6 is to process
    for i := 0; i < workers; i++ {
        go func() {
            for line := range lines {
                if matches := getRx.FindStringSubmatch(line);
                    matches != nil {
					//match[0]: entire matching text, [1]: text matched
					//  by parenthesized subexpression
                    pageMap.Update(matches[1], incrementer)
                }
            }
            done <- struct{}{}
        }()
    }
}

func waitUntil(done <-chan struct{}) {
	//blocked until all work done
    for i := 0; i < workers; i++ {
        <-done
    }
}

func showResults(pageMap safemap.SafeMap) {
    pages := pageMap.Close() //get the underlying map
    for page, count := range pages {
        fmt.Printf("%8d %s\n", count, page)
    }
}
