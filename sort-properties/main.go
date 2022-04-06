package main

import (
	"flag"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Properties struct{
	Headers []string
	Props	map[string]string
}

func ParseMarkdownFile(content string)(linesBeforeProperties []string,properties Properties,linesAfterProperties []string,err error){
	lines := strings.Split(content,"\n")
	linesBeforeProperties = make([]string,0,10)
	linesAfterProperties = make([]string,0,10)
	// linesBeforeProperties
	propertiesStartIndex := 0
	for i,line := range lines{
		if strings.Contains(line,"### Properties"){
			propertiesStartIndex = i
			break
		}
		linesBeforeProperties = append(linesBeforeProperties,line)
	}
	properties = Properties{
		Headers	: nil,
		Props	: make(map[string]string,10),
	}
	propertiesEndIndex := 0
	//properties
	for i:=propertiesStartIndex+1;i<len(lines);i++{
		// empty line
		if !strings.Contains(lines[i],"|"){
			if properties.Headers==nil{
				continue
			}
			propertiesEndIndex = i
			break
		}
		// header line
		if properties.Headers==nil{
			properties.Headers = strings.Split(lines[i],"|")
			continue
		}
		// border line
		if strings.Contains(lines[i],"--"){
			continue
		}
		// properties
		attributes := strings.Split(lines[i],"|")
		properties.Props[attributes[0]] = lines[i]
	}
	linesAfterProperties = lines[propertiesEndIndex:]
	return linesBeforeProperties,properties,linesAfterProperties,err
}

func WriteMarkdownFile(fileName string,linesBeforeProperties []string,properties Properties,linesAfterProperties []string)error{
	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE,0600)
	if err!=nil{
		return errors.Wrap(err,"failed to open file\n")
	}
	var builder strings.Builder
	builder.WriteString(strings.Join(linesBeforeProperties,"\n"))
	builder.WriteString("### Properties  \n")
	builder.WriteString(strings.Join(properties.Headers,"|")+"\n")
	boarders := make([]string,len(properties.Headers))
	for i:=0;i<len(properties.Headers);i++{
		boarders[i] = "------------"
	}
	builder.WriteString(strings.Join(boarders,"|")+"\n")

	propertyNames := make([]string,0,len(properties.Props))
	for name := range properties.Props{
		propertyNames = append(propertyNames,name)
	}
	sort.Strings(propertyNames)

	for _, propertyName := range propertyNames{
		builder.WriteString(properties.Props[propertyName])
		builder.WriteString("\n")
	}
	builder.WriteString(strings.Join(linesAfterProperties,"\n"))
	f.WriteString(builder.String())
	return nil
}

func main(){
	srcFile := flag.String("file","","read from local file")
	remoteURL := flag.String("url", "", "read from latest remote url")

	flag.Parse()

	var content string
	var fileName string
	if *srcFile!=""{
		rawContent, err := ioutil.ReadFile(filepath.Clean(*srcFile))
		if err!=nil{
			log.Printf("read srcFile %s failed: %v\n",*srcFile,err)
			return
		}
		content = string(rawContent)
		fileName = "new-"+filepath.Base(*srcFile)
	} else if *remoteURL!=""{
		resp, err := http.Get(*remoteURL)
		if err!=nil{
			log.Printf("failed to get latest form:%v\n",err)
			return
		}
		defer resp.Body.Close()
		buf := new(strings.Builder)
		if _, err := io.Copy(buf, resp.Body);err!=nil{
			log.Printf("failed to read latest form:%v\n",err)
			return
		}
		content = buf.String()
		parts := strings.Split(*remoteURL,"/")
		fileName = parts[len(parts)-1]
	}else{
		log.Printf("usage: --file or --url\n")
		return
	}

	linesBeforeProperties,properties,linesAfterProperties, err := ParseMarkdownFile(content)
	if err!=nil{
		log.Printf("parse markdown file failed: %v\n", err)
	}
	if err = WriteMarkdownFile(fileName,linesBeforeProperties,properties,linesAfterProperties);err!=nil{
		log.Printf("write markdown file failed: %v\n",err)
	}
}