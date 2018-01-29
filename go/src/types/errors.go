package types

var malError *Data = nil

func Throw(val *Data) *Data {
	malError = val
	return nil
}

func HasError() bool {
	return malError != nil
}

func GetError() *Data {
	return malError
}

func ClearError() {
	malError = nil
}
