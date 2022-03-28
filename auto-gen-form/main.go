package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

const (
	TestCaseGitURL = "https://github.com/oam-dev/catalog"
	FormName = "cloud-resources-list.md"
	FormEnUrl = "https://raw.githubusercontent.com/oam-dev/kubevela.io/main/docs/end-user/components/cloud-services/cloud-resources-list.md"
)

func ReadTestCases() ([]string, error) {
	tmpPath := "./tmp"
	//if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
	//	err := os.RemoveAll(tmpPath)
	//	if err != nil {
	//		return nil, errors.Wrap(err, "failed to remove the directory")
	//	}
	//}
	//_, err := git.PlainClone(tmpPath, false, &git.CloneOptions{
	//	URL:      TestCaseGitURL,
	//	Progress: nil,
	//})
	//if err != nil {
	//	return nil, err
	//}
	basePath := filepath.Join(tmpPath, "catalog/addons")
	infos, err := ioutil.ReadDir(filepath.Join(tmpPath, "catalog/addons"))
	if err != nil {
		return nil, err
	}

	res := make([]string, 0, 10)
	for _, info := range infos {
		var newName = info.Name()
		if strings.HasPrefix(newName, "terraform-") {
			dirPath := filepath.Join(basePath,newName+"/definitions")
			definitions,err := ioutil.ReadDir(dirPath)
			if err!=nil{
				log.Printf("ReadDir %s with errors: %v",dirPath,err)
				continue
			}
			for _, definition := range definitions{
				var definitionName = definition.Name()
				if !strings.HasSuffix(definitionName, ".yaml"){
					continue
				}
				content, err := ioutil.ReadFile(filepath.Clean(filepath.Join(dirPath, definitionName)))
				if err!=nil{
					log.Printf("Open file %s with errors: %v",filepath.Join(dirPath, definitionName),err)
					continue
				}
				if strings.Contains(string(content),"definition.oam.dev/verified: \"true\""){
					testCaseName := strings.TrimSuffix(definitionName,".yaml")
					res = append(res,strings.TrimPrefix(testCaseName,"terraform-"))
				}
			}
		}
	}
	return res, nil
}

func UpdateMarkdownForm(ValidTestCases []string) error{
	// read original form file
	resp, err := http.Get(FormEnUrl)
	if err!=nil{
		return errors.Wrap(err, "failed to get latest form")
	}
	defer resp.Body.Close()
	buf := new(strings.Builder)
	if _, err := io.Copy(buf, resp.Body);err!=nil{
		return errors.Wrap(err, "failed to read latest form")
	}
	originFormContent := buf.String()

	addValid := false
	//edit form file
	lines := strings.Split(originFormContent,"\n")
	for index,line := range lines{
		// header
		if strings.Contains(line, "Orchestration Type"){
			if strings.Contains(line, "Valid"){
				continue
			}
			lines[index] = fmt.Sprintf("%s Valid |",line)
			addValid = true
			continue
		}

		// border
		if strings.Contains(line, "|-"){
			if addValid{
				lines[index] = fmt.Sprintf("%s-----------|",line)
			}
			continue
		}

		// row
		if strings.Contains(line, "|"){
			columns := strings.Split(line, "|")
			if !((addValid && len(columns)>=6) || (!addValid && len(columns)>=7)){
				log.Printf("invalid number of columns %v %v\n",len(columns),columns)
				return errors.Wrap(err, "invalid number of columns")
			}

			if addValid {
				lines[index] = fmt.Sprintf("%s %v |",line,MatchTestCaseName(columns[3], ValidTestCases))
				continue
			}

			columns[len(columns)-1] = fmt.Sprintf(" %v ",MatchTestCaseName(columns[3], ValidTestCases))
			lines[index] = strings.Join(columns,"|")
			continue
		}
	}

	var builder strings.Builder
	for _,line := range lines{
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	newFormContent := builder.String()

	// write local form file
	markdownFile := FormName
	f, err := os.OpenFile(filepath.Clean(markdownFile), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", markdownFile, err)
	}
	if err = os.Truncate(markdownFile, 0); err != nil {
		return fmt.Errorf("failed to truncate file %s: %w", markdownFile, err)
	}
	if _, err := f.WriteString(newFormContent); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return nil
}

func MatchTestCaseName(ResourceReference string, ValidTestCases []string)bool{
	parts := strings.Split(ResourceReference,"/")
	if len(parts)==0{
		return false
	}
	fileName := parts[len(parts)-1]
	TestCaseName := strings.Split(fileName,".")[0]
	log.Println(TestCaseName)
	for _,ValidTestCase := range ValidTestCases{
		if TestCaseName==ValidTestCase{
			return true
		}
	}
	return false
}

func main(){
	ValidTestCases,err := ReadTestCases()
	if err!=nil{
		log.Print(errors.Wrap(err,"read test cases failed"))
	}
	log.Printf("%+v\n", ValidTestCases)
	if err = UpdateMarkdownForm(ValidTestCases);err!=nil{
		log.Print(errors.Wrap(err,"update markdown failed"))
	}
}
