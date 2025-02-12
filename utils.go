package meetup

import (
	"errors"

	"github.com/jaevor/go-nanoid"
)

var customChars = [62]byte{
	'A', 'B', 'C', 'D', 'E',
	'F', 'G', 'H', 'I', 'J',
	'K', 'L', 'M', 'N', 'O',
	'P', 'Q', 'R', 'S', 'T',
	'U', 'V', 'W', 'X', 'Y',
	'Z', 'a', 'b', 'c', 'd',
	'e', 'f', 'g', 'h', 'i',
	'j', 'k', 'l', 'm', 'n',
	'o', 'p', 'q', 'r', 's',
	't', 'u', 'v', 'w', 'x',
	'y', 'z', '0', '1', '2',
	'3', '4', '5', '6', '7',
	'8', '9',
}

func FlattenErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	errString := ""
	for _, err := range errs {
		errString += err.Error() + "\n"
	}

	return errors.New(errString)
}

func GenerateID(length int) string {
	canonicID, err := nanoid.CustomASCII(string(customChars[:]), length)
	if err != nil {
		panic(err)
	}
	return canonicID()
}
