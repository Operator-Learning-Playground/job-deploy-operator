package common

import "os"

func GetWd() string {
	wd := os.Getenv("WORK_DIR")
	if wd == "" {
		wd, _ = os.Getwd()
	}
	return wd
}

// CreateByJobTemplate 给 Job 资源对象打的 Annotation, 用于
// 标示使用此 JobTemplate 的 Job 资源
// ex: annotation: createByJobTemplate: default.job-template-1
const CreateByJobTemplate = "createByJobTemplate"

func GetConnectionOfJobAndJobTemplate(namespace, name string) string {
	return namespace + "." + name
}
