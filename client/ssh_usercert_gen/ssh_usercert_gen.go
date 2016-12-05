package main

import (
	"bytes"
	//"crypto/tls"
	"errors"
	//"flag"
	//"fmt"
	//"github.com/prometheus/client_golang/prometheus"
	//"gopkg.in/ldap.v2"
	//"gopkg.in/yaml.v2"
	//"io"
	"io/ioutil"
	"log"
	//"net"
	//"net/http"
	"os"
	"os/exec"
	"regexp"
	//"strconv"
	"strings"
	//"sync"
	//"time"
)

func getUserPubKey(username string) (string, error) {
	cmd := exec.Command("/usr/bin/sss_ssh_authorizedkeys", username)
	cmd.Stdin = strings.NewReader("some input")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	log.Printf("Pub key: %s\n", out.String())
	return out.String(), nil
}

// gen_user_cert a username and key, returns a short lived cert for that user
func gen_cert_internal(username string, userPubKey string, users_ca_filename string, host_identity string) (string, error) {

	//Convert userKey into temp file
	content := []byte(userPubKey)
	tmpfile, err := ioutil.TempFile("/tmp/", "userkey")
	if err != nil {
		log.Fatal(err)
	}
	defer tmpfile.Close()
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write(content); err != nil {
		log.Fatal(err)
	}

	keyIdentity := host_identity + "_" + username

	cmd := exec.Command("ssh-keygen", "-s", users_ca_filename, "-I", keyIdentity, "-n", username, "-V", "+1d", tmpfile.Name())
	cmd.Stdin = strings.NewReader("\n")
	var out bytes.Buffer
	cmd.Stdout = &out

	var cmderr bytes.Buffer
	cmd.Stderr = &cmderr
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("stdout: %q\n", out.String())
	log.Printf("stderr: %q\n", cmderr.String())

	//Signed user key /tmp/userkey322296953-cert.pub: id "foo" serial 0 for bar valid from 2016-12-05T21:38:00 to 2016-12-06T19:39:45
	re := regexp.MustCompile("^Signed user key ([^:]+):")
	match := re.FindStringSubmatch(cmderr.String())
	if len(match) != 2 {
		log.Printf("badmatch; %v\n", match)
		err := errors.New("cannot find signed key name, re find failure")
		return "", err
	}
	outFilename := match[1]
	log.Printf("outfilename: %v\n", outFilename)
	defer os.Remove(outFilename)

	fileBytes, err := ioutil.ReadFile(outFilename)
	if err != nil {
		return "", err
	}

	return string(fileBytes[:]), nil
}

func genUserCert(userName string, users_ca_filename string) (string, error) {

	userPubKey, err := getUserPubKey(userName)
	if err != nil {
		log.Fatal(err)
	}
	hostIdentity := "mon-sre-dev"
	cert, err := gen_cert_internal(userName, userPubKey, users_ca_filename, hostIdentity)
	if err != nil {
		log.Fatal(err)
	}
	return cert, err
}

func main() {
	cert, err := genUserCert("camilo_viecco1", "/home/camilo_viecco1/ssh-test/users_ca")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("cert=%s", cert)
}
