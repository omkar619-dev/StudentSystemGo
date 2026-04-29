package handlers

import(
"errors"
"reflect"
"strings"
"restapi/pkg/utils"
)

func CheckBlankFields(teacher interface{}) error {
	val := reflect.ValueOf(teacher)
	for i := 0; i < val.NumField(); i++ {
		fieldValue := val.Field(i)
		if fieldValue.Kind() == reflect.String && fieldValue.String() == "" {
			// http.Error(w, "Missing required fields in request body", http.StatusBadRequest)
			return utils.ErrorHandler(errors.New("Missing required fields in request body"), "Missing required fields in request body")
		}
	}
	return nil
}

func GetFieldNames(model interface{}) []string {
	val := reflect.TypeOf(model)
	fields := []string{}
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldToAdd := strings.TrimSuffix(field.Tag.Get("json"), ",omitempty")
		fields = append(fields, fieldToAdd) // get json tag
	}
	return fields
}

