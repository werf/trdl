package util

import (
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func CheckRequiredFields(req *logical.Request, fields *framework.FieldData) error {
	for fieldName, schema := range fields.Schema {
		if schema.Required && req.Get(fieldName) == nil {
			return fmt.Errorf("required field %q must be set", fieldName)
		}
	}

	return nil
}
