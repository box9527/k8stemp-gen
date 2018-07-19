package processor

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"text/template"

	yaml "gopkg.in/yaml.v2"
)

// Vortex container of information that are awesome and amazing
type Vortex struct {
	variables map[string]interface{}
	strict    bool
}

func New() *Vortex {
	return &Vortex{}
}

// Set allows the user to define variables as command line arguments
func (v *Vortex) Set(input string) error {
	data := strings.Split(input, "=")
	// If we don't have a key value pair split by = then reject it
	if len(data) != 2 {
		return errors.New("Incorrect format, expect key=value")
	}
	v.variables[data[0]] = data[1]
	return nil
}

func (v *Vortex) String() string {
	return fmt.Sprintf("Vortex has %v loaded", len(v.variables))
}

// LoadVariables will read from a file path and load Vortex with the variables ready
func (v *Vortex) LoadVariables(variablepath string) error {
	if _, err := os.Stat(variablepath); os.IsNotExist(err) {
		return fmt.Errorf("%v is not a valid path", variablepath)
	}
	buff, err := ioutil.ReadFile(variablepath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(buff, &(v.variables))
}

func (v *Vortex) EnableStrict() *Vortex {
	v.strict = true
	return v
}

// ProcessTemplates applys a DFS over the templateroot and will process the
// templates with the stored vortex variables
func (v *Vortex) ProcessTemplates(templateroot, outputroot string) error {
	// If the folder path doesn't exist, then say so
	// If the templateroot is a file, just process that
	root, err := os.Stat(templateroot)
	if os.IsNotExist(err) {
		return fmt.Errorf("%v does not exist", templateroot)
	}
	if !root.IsDir() {
		return v.processTemplate(templateroot, outputroot)
	}
	files, err := ioutil.ReadDir(templateroot)
	if err != nil {
		return err
	}
	for _, file := range files {
		readpath := path.Join(templateroot, file.Name())
		switch {
		// Ensure we don't automatically recurse down hidden files
		case file.IsDir() && !strings.HasPrefix(file.Name(), "."):
			newroot := path.Join(outputroot, file.Name())
			if err := v.ProcessTemplates(readpath, newroot); err != nil {
				return err
			}
		default:
			// If the file extension doesn't match what we expect then ignore it
			if err = v.processTemplate(readpath, outputroot); err != nil {
				return err
			}
		}
	}
	return nil
}

func (v *Vortex) processTemplate(templatepath, outputpath string) error {
	if !strings.HasSuffix(templatepath, ".yaml") {
		return nil
	}
	// if the folder path doesn't exist, then we need to make it
	// and make sure we don't create a directory if we are just validating the contents
	if _, err := os.Stat(outputpath); os.IsNotExist(err) && !v.strict {
		if err = os.MkdirAll(outputpath, 0755); err != nil {
			return err
		}
	}
	if f, err := os.Stat(outputpath); !os.IsNotExist(err) && !f.IsDir() {
		return fmt.Errorf("%v already exists, needs to be removed in order to process", outputpath)
	}
	buff, err := ioutil.ReadFile(templatepath)
	if err != nil {
		return err
	}
	tmpl, err := template.New(path.Base(templatepath)).Parse(string(buff))
	if err != nil {
		return err
	}
	if v.strict {
		tmpl = tmpl.Option("missingkey=error")
	}
	writer := bytes.NewBuffer(nil)
	if err = tmpl.Execute(writer, v.variables); err != nil {
		return err
	}

	// Don't write the file if we have been told to validate only
	if !v.strict {
		filename := path.Join(outputpath, path.Base(templatepath))
		return ioutil.WriteFile(filename, writer.Bytes(), 0644)
	}
	// ensure that we have a valid yaml file at the end of it
	return yaml.UnmarshalStrict(writer.Bytes(), map[string]interface{}{})
}
