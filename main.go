package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

    "github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/sirupsen/logrus"
)

type passEntry struct {
    pass, user, domain, name, desc string
}

type config struct {
	address, mainPassword, privateKeyPath string
}

func main() {
    var p passEntry
	var c config
	flag.StringVar(&p.domain, "uri", "", "Domain on which the password will be used")
	flag.StringVar(&p.name, "name", "Example Account", "Password name")
	flag.StringVar(&p.desc, "desc", "Password added with pass2passbolt", "Password description")
	flag.StringVar(&c.address, "address", "https://passbolt.local", "Passbolt server address")
	flag.StringVar(&c.mainPassword, "main-password", "", "User main password")
	flag.StringVar(&c.privateKeyPath, "private-key", "", "Path to user's private key")
    flag.Parse()

	if errs := validateConfig(c); len(errs) != 0 {
		for _, err := range errs {
			logrus.Error(err)
		}
		flag.Usage()
		logrus.Fatal("too many missing arguments")
	}

    pipe, err := isInputFromPipe()
    if err != nil {
        logrus.Fatalf("cannot detect input method: %s", err)
    }

    if !pipe {
        // read from stdin
        fmt.Println("cannot read from stdin for now")
        return
    }

	key, err := os.ReadFile(c.privateKeyPath)
	if err != nil {
		logrus.Fatalf("cannot read private key file: %s", err)
	}

	ctx := context.TODO()

	// create the passbolt client
	client, err := api.NewClient(nil, "", c.address, string(key), c.mainPassword)
	if err != nil {
		logrus.Fatalf("cannot create passbolt client: %s", err)
	}

    reader := bufio.NewReader(os.Stdin)
    s := bufio.NewScanner(reader)

    // with pass, the first line is always the password, so we can get it here
    s.Scan()
    p.pass = s.Text()

    // then we can try to parse the others field
    for s.Scan() {
        if strings.HasPrefix(s.Text(), "user:") {
            p.user = s.Text()[6:] // trim the prefix
        }
    }

    fmt.Printf("domain: %s\nuser: %s\npass: %s\n", p.domain, p.user, p.pass)

	if err = client.Login(ctx); err != nil {
		logrus.Fatalf("cannot login to passbolt: %s", err)
	}

    resourceID, err := helper.CreateResource(
		ctx,
		client,
		"",
		"Example Account",
		p.user,
		p.domain,
		p.pass,
		"This is an Account for the example test portal",
	)

	if err != nil {
		logrus.Fatalf("cannot create resource: %s", err)
	}

	fmt.Printf("resouce %s created!\n", resourceID)
}

func isInputFromPipe() (bool, error) {
    fileInfo, err := os.Stdin.Stat()
    return fileInfo.Mode() & os.ModeCharDevice == 0, err
}

func validateConfig(c config) []error {
	var errs []error
	if c.mainPassword == "" {
		errs = append(errs, errors.New("main password is missing"))
	}

	if c.privateKeyPath == "" {
		errs = append(errs, errors.New("path to private key file is missing"))
	} else {
		if _, err := os.Stat(c.privateKeyPath); errors.Is(err, os.ErrNotExist) {
			errs = append(errs, errors.New(
				fmt.Sprintf("path %s does not exist", c.privateKeyPath)))
		}
	}

	return errs
}
