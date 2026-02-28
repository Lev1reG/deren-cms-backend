package response

import (
	"encoding/json"
	"fmt"
	"encore.dev/beta/errs"
)

// Just to print what ErrCode marshals to
func PrintErrCodeFormat(code errs.ErrCode) {
	b, _ := json.Marshal(code)
	fmt.Printf("ErrCode %s marshals to: %s\n", code.String(), string(b))
}
