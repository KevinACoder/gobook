// Copyright © 2010-12 Qtrac Ltd.
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
    "io/ioutil"
    "log"
    "os"
    "path/filepath"
    "regexp"
    "strings"
)

var britishAmerican = "british-american.txt"

func init() {
    dir, _ := filepath.Split(os.Args[0])
    britishAmerican = filepath.Join(dir, britishAmerican)
}

func main() {
    inFilename, outFilename, err := filenamesFromCommandLine()
    if err != nil {
        fmt.Println(err)
        os.Exit(1) //any pending defer func will lose
    }
	//default file descriptor
    inFile, outFile := os.Stdin, os.Stdout
    if inFilename != "" {
        if inFile, err = os.Open(inFilename); err != nil {
			//log the error and exit
            log.Fatal(err)
        }
        defer inFile.Close()
    }
    if outFilename != "" {
		//create or truncate
        if outFile, err = os.Create(outFilename); err != nil {
            log.Fatal(err)
        }
        defer outFile.Close()
    }

    if err = americanise(inFile, outFile); err != nil {
        log.Fatal(err)
    }
}

func filenamesFromCommandLine() (inFilename, outFilename string,
    err error) {
    if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
        err = fmt.Errorf("usage: %s [<]infile.txt [>]outfile.txt",
            filepath.Base(os.Args[0]))
        return "", "", err
    }
    if len(os.Args) > 1 {
        inFilename = os.Args[1]
        if len(os.Args) > 2 {
            outFilename = os.Args[2]
        }
    }
    if inFilename != "" && inFilename == outFilename {
        log.Fatal("won't overwrite the infile")
    }
    return inFilename, outFilename, nil
}

func americanise(inFile io.Reader, outFile io.Writer) (err error) {
    reader := bufio.NewReader(inFile) //buffer the in\out
    writer := bufio.NewWriter(outFile)
	//flush writer buffer before leave
    defer func() {
        if err == nil {
            err = writer.Flush() //if flush is fail, err flag would be returned to caller
        }
    }()

    var replacer func(string) string //1
    if replacer, err = makeReplacerFunction(britishAmerican); err != nil {
        return err
    }
	//panic if the regex is invalid
    wordRx := regexp.MustCompile("[A-Za-z]+")
    eof := false
    for !eof {
        var line string //2
        line, err = reader.ReadString('\n') //read lines from buffered reader
        if err == io.EOF {
            err = nil   // io.EOF isn't really an error
            eof = true  // this will end the loop at the next iteration
        } else if err != nil { //3
            return err  // finish immediately for real errors
        }
		//replace every matches
        line = wordRx.ReplaceAllStringFunc(line, replacer)
		//write to buffered writer
        if _, err = writer.WriteString(line); err != nil {
            return err
        }
    }
    return nil
}

/*input path of file include word to convert, return a replacer and error flag*/
func makeReplacerFunction(file string) (func(string) string, error) {
    rawBytes, err := ioutil.ReadFile(file)
    if err != nil {
        return nil, err
    }
    text := string(rawBytes)

    usForBritish := make(map[string]string)
    lines := strings.Split(text, "\n") //split string by line
    for _, line := range lines {
        fields := strings.Fields(line) //tokenize string by whitespace with some forgiving for human edited file
        if len(fields) == 2 {
            usForBritish[fields[0]] = fields[1]
        }
    }

    return func(word string) string {
		//lookup table, the reference can be passed around
        if usWord, found := usForBritish[word]; found {
            return usWord
        }
        return word
    }, nil
}
