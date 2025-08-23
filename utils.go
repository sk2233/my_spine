package main

func HandleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func Use(args ...any) {}
