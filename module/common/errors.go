package common

func JustError(_ any, err error) error {
	return err
}

func IgnoreError[Any any](any Any, _ error) Any {
	return any
}

func IgnoreBoolean[Any any](any Any, _ bool) Any {
	return any
}
