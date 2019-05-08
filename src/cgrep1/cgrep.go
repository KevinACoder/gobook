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

// The approach taken here was inspired by an example on the gonuts mailing
// list by Roger Peppe.

package main

import (
    "bufio"
    "bytes"
    "fmt"
    "io"
    "log"
    "os"
    "path/filepath"
    "regexp"
    "runtime"
)

var workers = runtime.NumCPU()

type Result struct {
    filename string  //file name
    lino     int	 //line number
    line     string  //string content of line
}

type Job struct {
    filename string  //the name of the file on procesing
    results  chan<- Result  //channel that any result to be sent
}

func (job Job) Do(lineRx *regexp.Regexp) {
    file, err := os.Open(job.filename)
    if err != nil {
        log.Printf("error: %s\n", err)
        return
    }
    defer file.Close()
    reader := bufio.NewReader(file) //read file into buf
    for lino := 1; ; lino++ {
        line, err := reader.ReadBytes('\n') //read a line at one time
        line = bytes.TrimRight(line, "\n\r") //get rid of line end
        if lineRx.Match(line) {
			//get one match result and sent to result channel
            job.results <- Result{job.filename, lino, string(line)}
        }
        if err != nil {
            if err != io.EOF {
                log.Printf("error:%d: %s\n", lino, err)
            }
            break //reach EOF or other error
        }
    }
}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU()) // Use all the machine's cores
    if len(os.Args) < 3 || os.Args[1] == "-h" || os.Args[1] == "--help" {
        //print prompt info
		fmt.Printf("usage: %s <regexp> <files>\n",
            filepath.Base(os.Args[0]))
        os.Exit(1)
    }

	//Compile the input regex
	// lineRx is a shared pointer to value, which shall be a cause of 
	//  concern since it's not thread safe, but Go doc *regexp.Regexp is
	//  safe to be shared in as many routines
    if lineRx, err := regexp.Compile(os.Args[1]); err != nil {
        log.Fatalf("invalid regexp: %s\n", err)
    } else {
		//regex + file list
        grep(lineRx, commandLineFiles(os.Args[2:]))
    }
}

func commandLineFiles(files []string) []string {
    if runtime.GOOS == "windows" {
        args := make([]string, 0, len(files))
        for _, name := range files {
			//return names of all files matching pattern
            if matches, err := filepath.Glob(name); err != nil {
                args = append(args, name) // Invalid pattern
            } else if matches != nil { // At least one match
                args = append(args, matches...)
            }
        }
        return args
    }
    return files
}

func grep(lineRx *regexp.Regexp, filenames []string) {
	//create three bidirection channel as per processor
	//  therefore to minimize needless blocking
    jobs := make(chan Job, workers)
	//results is implemented as a much larger buffer
    results := make(chan Result, minimum(1000, len(filenames)))
    done := make(chan struct{}, workers) //not caring whether it's true or flase

    go addJobs(jobs, filenames, results) // Executes in its own goroutine
    //do jobs in four seprate routines (executions)
	for i := 0; i < workers; i++ {
        go doJobs(done, lineRx, jobs) // Each executes in its own goroutine
    }
	//wait all work to be done and close the results channel
    go awaitCompletion(done, results) // Executes in its own goroutine
    processResults(results)           // Blocks until the work is done
}

func addJobs(jobs chan<- Job, filenames []string, results chan<- Result) {
    //send every job to job channel, the job channel has a buffer size of
	//  four, so the first four jobs are executed immediately, other are 
	//  waiting until free up room
	for _, filename := range filenames {
        jobs <- Job{filename, results} //send job to channel
    }
    close(jobs) //close after all jobs have been sent
}

func doJobs(done chan<- struct{}, lineRx *regexp.Regexp, jobs <-chan Job) {
    //each execution iter over the same receive-only channel
	for job := range jobs {
		//routine blocked until job is available
        job.Do(lineRx)
    }
    done <- struct{}{} //signify until run out of jobs
}

func awaitCompletion(done <-chan struct{}, results chan Result) {
    for i := 0; i < workers; i++ {
        <-done //blocking until all execution send signified
    }
    close(results)
}

func processResults(results <-chan Result) {
	//blocking waiting for results
    for result := range results {
        fmt.Printf("%s:%d:%s\n", result.filename, result.lino, result.line)
    }
	//once all results have been processed, loop finished and program
	//  terminate
}

func minimum(x int, ys ...int) int {
    for _, y := range ys {
        if y < x {
            x = y
        }
    }
    return x
}
