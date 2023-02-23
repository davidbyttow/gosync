package async

import "errors"

func joinErrors(errs ...error) error {
	return errors.Join(errs...)
}
