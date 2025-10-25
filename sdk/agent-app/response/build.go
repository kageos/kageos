package response

import "github.com/pkg/errors"

func build(resp *RunFunctionResp, data interface{}) error {
	if resp == nil {
		return errors.New("resp is nil")
	}
	//resp.Data = data
	return nil
}
