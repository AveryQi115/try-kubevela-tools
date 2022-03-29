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
const(
	generateFileName = "cloud-resources-list.md"
)

func WriteMarkdownFile(title string, header []string, rows map[string]map[string]string)error{
	fileName := generateFileName
	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE,0600)
	if err!=nil{
		return errors.Wrap(err,"failed to open file\n")
	}
	var builder strings.Builder
	// title
	builder.WriteString("---\n")
	builder.WriteString(title)
	builder.WriteString("\n")
	builder.WriteString("---\n \n")

	// header
	builder.WriteString("|"+strings.Join(header,"|")+"|\n")
	builder.WriteString("|--------------------|-----------------------|-----------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------|\n")

	// rows
	providers := make([]string,0,5)
	for provider := range rows{
		providers = append(providers,provider)
	}
	sort.Strings(providers)
	for provider_index, provider := range providers{
		resources := rows[provider]
		resourceNames := make([]string,0,10)
		for resource := range resources{
			resourceNames = append(resourceNames,resource)
		}
		sort.Strings(resourceNames)
		for resource_index, resourceName := range resourceNames{
			if resource_index==0{
				line := resources[resourceName]
				columns := strings.Split(line,"|")
				columns = columns[1:len(columns)-1]
				if provider_index==0{
					columns[0]="Terraform"
				}
				columns[1]=provider
				resources[resourceName]="|"+strings.Join(columns,"|")+"|"
			}
			builder.WriteString(resources[resourceName])
			builder.WriteString("\n")
		}
	}
	f.WriteString(builder.String())
	return nil
}

func ParseMarkdownFile(content string)(title string,header []string,rows map[string]map[string]string, err error){
	lines := strings.Split(content,"\n")
	var provider string
	rows = make(map[string]map[string]string)
	for _,line := range lines{
		// title
		if strings.Contains(line, "title"){
			title = line
			continue
		}

		// header
		if strings.Contains(line, "Orchestration Type") || strings.Contains(line, "编排类型"){
			splits := strings.Split(line,"|")
			header = splits[1:len(splits)-1]
			if len(header)!=4 && len(header)!=5{
				log.Printf("header %s: invalid column number %v\n",line,len(header))
				return title,header,nil,errors.Errorf("Invalid column number: %v",len(header))
			}
			continue
		}

		// row
		if strings.Contains(line, "|") && !strings.Contains(line, "---"){
			columns := strings.Split(line, "|")
			columns = columns[1:len(columns)-1]
			curProvider := strings.TrimLeft(columns[1]," ")
			curProvider = strings.TrimRight(curProvider," ")
			if len(curProvider)!=0{
				provider = curProvider
			}
			if len(provider)==0{
				log.Printf("line %s: no valid provider name %v\n",line,columns[1])
				return title,header,nil,errors.Errorf("line %s: no valid provider name %v\n",line,columns[1])
			}

			begin := strings.Index(columns[2],"[")
			end := strings.Index(columns[2],"]")
			resourceName := columns[2][begin+1:end]
			if val,exist := rows[provider];exist{
				columns[0] = "                       "
				columns[1] = "                       "
				val[resourceName]="|"+strings.Join(columns,"|")+"|"
			}else{
				rows[provider]=make(map[string]string)
				columns[0] = "                       "
				columns[1] = "                       "
				rows[provider][resourceName] = "|"+strings.Join(columns,"|")+"|"
			}
		}
	}
	return title, header, rows, nil
}

func main(){
	srcFile := flag.String("file","","read from local file")
	remoteURL := flag.String("url", "", "read from latest remote url")

	flag.Parse()

	var content string
	if *srcFile!=""{
		rawContent, err := ioutil.ReadFile(filepath.Clean(*srcFile))
		if err!=nil{
			log.Printf("read srcFile %s failed: %v\n",*srcFile,err)
			return
		}
		content = string(rawContent)
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
	}

	title, header, rows, err := ParseMarkdownFile(content)
	if err!=nil{
		log.Printf("parse markdown file failed: %v\n", err)
	}
	log.Printf("%s\n",title)
	log.Printf("%s\n",header)
	if err = WriteMarkdownFile(title,header,rows);err!=nil{
		log.Printf("write markdown file failed: %v\n",err)
	}
}