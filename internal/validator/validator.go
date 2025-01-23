package validator

import (
    "encoding/json"
    "fmt"
    "reflect"
    "regexp"
    "strings"
)

var (
    emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
    symbolRegex   = regexp.MustCompile(`^[A-Z]{1,5}$`)
    passwordRegex = regexp.MustCompile(`^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]{8,}$`)
)

type Validator struct {
    Errors map[string]string
}

func New() *Validator {
    return &Validator{Errors: make(map[string]string)}
}

func (v *Validator) Valid() bool {
    return len(v.Errors) == 0
}

func (v *Validator) AddError(key, message string) {
    if _, exists := v.Errors[key]; !exists {
        v.Errors[key] = message
    }
}

func (v *Validator) Check(ok bool, key, message string) {
    if !ok {
        v.AddError(key, message)
    }
}

func (v *Validator) ValidateEmail(email string) {
    v.Check(emailRegex.MatchString(email), "email", "must be a valid email address")
}

func (v *Validator) ValidatePassword(password string) {
    v.Check(passwordRegex.MatchString(password), "password", 
        "must be at least 8 characters and contain uppercase, lowercase, number and special character")
}

func (v *Validator) ValidateSymbol(symbol string) {
    v.Check(symbolRegex.MatchString(symbol), "symbol", "must be 1-5 uppercase letters")
}

func (v *Validator) ValidateJSON(data []byte, target interface{}) error {
    if err := json.Unmarshal(data, target); err != nil {
        return fmt.Errorf("invalid JSON format: %v", err)
    }

    val := reflect.ValueOf(target).Elem()
    typ := val.Type()

    for i := 0; i < val.NumField(); i++ {
        field := val.Field(i)
        fieldType := typ.Field(i)
        
        tags := strings.Split(fieldType.Tag.Get("validate"), ",")
        for _, tag := range tags {
            switch tag {
            case "required":
                if field.IsZero() {
                    v.AddError(fieldType.Name, "field is required")
                }
            case "email":
                if email, ok := field.Interface().(string); ok {
                    v.ValidateEmail(email)
                }
            case "symbol":
                if symbol, ok := field.Interface().(string); ok {
                    v.ValidateSymbol(symbol)
                }
            }
        }

        if min := fieldType.Tag.Get("min"); min != "" {
            v.validateMin(field, min, fieldType.Name)
        }
        if max := fieldType.Tag.Get("max"); max != "" {
            v.validateMax(field, max, fieldType.Name)
        }
    }

    return nil
}

func (v *Validator) validateMin(field reflect.Value, min, fieldName string) {
    switch field.Kind() {
    case reflect.String:
        v.Check(len(field.String()) >= parseInt(min), fieldName, 
            fmt.Sprintf("must be at least %s characters", min))
    case reflect.Int, reflect.Int64:
        v.Check(field.Int() >= int64(parseInt(min)), fieldName,
            fmt.Sprintf("must be at least %s", min))
    case reflect.Float64:
        v.Check(field.Float() >= float64(parseInt(min)), fieldName,
            fmt.Sprintf("must be at least %s", min))
    }
}

func (v *Validator) validateMax(field reflect.Value, max, fieldName string) {
    switch field.Kind() {
    case reflect.String:
        v.Check(len(field.String()) <= parseInt(max), fieldName,
            fmt.Sprintf("must not exceed %s characters", max))
    case reflect.Int, reflect.Int64:
        v.Check(field.Int() <= int64(parseInt(max)), fieldName,
            fmt.Sprintf("must not exceed %s", max))
    case reflect.Float64:
        v.Check(field.Float() <= float64(parseInt(max)), fieldName,
            fmt.Sprintf("must not exceed %s", max))
    }
}

func parseInt(s string) int {
    var n int
    fmt.Sscanf(s, "%d", &n)
    return n
}