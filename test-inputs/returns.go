package main

type Foobar struct{}

func naked() error {
	return nil
}

func nakedPtr() *string {
	return nil
}

func nakedStruct() Foobar {
	return Foobar{}
}

func nakedTuple() (string, error) {
	return "", nil
}

func nakedTuplePtr() (*string, error) {
	return nil, nil
}

func named() (err error) {
	return nil
}

func namedPtr() (s *string) {
	return nil
}

func namedTuple() (s string, err error) {
	return "", nil
}

func namedTuplePtr() (s *string, err error) {
	return nil, nil
}
