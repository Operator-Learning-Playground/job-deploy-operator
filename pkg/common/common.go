package common

import "os"

func GetWd() string {
	wd := os.Getenv("WORK_DIR")
	if wd == "" {
		wd, _ = os.Getwd()
	}
	return wd
}

// ex: annotation: createByJobTemplate: default.job-template-1

const CreateByJobTemplate = "createByJobTemplate"

func GetConnectionOfJobAndJobTemplate(namespace, name string) string {
	return namespace + "." + name
}
