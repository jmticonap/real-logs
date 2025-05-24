package test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshelingUndefinedField(t *testing.T) {
	t.Run("Unmarshal with undefined int field", func(t *testing.T) {
		type Person struct {
			Firstname string `json:"firstname"`
			Lastname  string `json:"lastname"`
			Age       int    `json:"age"`
		}
		strJson := `
			{
				"firstname": "Donovan",
				"lastname":"Rey"
			}
		`
		var person Person
		err := json.Unmarshal([]byte(strJson), &person)

		assert.NoError(t, err, "No debería retornar error")
		assert.Equalf(t, person.Age, 0, "El valor no es el esperado | 0")
	})

	t.Run("Unmarshal with undefined string field", func(t *testing.T) {
		type Person struct {
			Firstname string `json:"firstname"`
			Lastname  string `json:"lastname"`
			Age       int    `json:"age"`
		}
		strJson := `
			{
				"firstname": "Donovan",
				"age": 10
			}
		`
		var person Person
		err := json.Unmarshal([]byte(strJson), &person)

		assert.NoError(t, err, "No debería retornar error")
		assert.Equalf(t, person.Lastname, "", "El valor no es el esperado | ''")
	})
}
