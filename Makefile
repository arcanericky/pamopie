EXECUTABLE=bin/pam_opie.so
GOSRCS=authenticate.go pamcgo.go

$(EXECUTABLE): $(GOSRCS) converse.o
	go build -o $(EXECUTABLE) --buildmode=c-shared $(GOSRCS)
	rm -f bin/pam_opie.h

converse.o: converse/converse.c
	gcc -c converse/converse.c

test:
	go test -race -coverprofile=coverage.txt -covermode=atomic authenticate.go authenticate_test.go
	go tool cover -html=coverage.txt -o coverage.html

clean:
	rm -rf bin converse.o coverage.*
