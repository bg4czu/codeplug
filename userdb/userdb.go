// Copyright 2017-2018 Dale Farnsworth. All rights reserved.

// Dale Farnsworth
// 1007 W Mendoza Ave
// Mesa, AZ  85210
// USA
//
// dale@farnsworth.org

// This file is part of UserDB.
//
// UserDB is free software: you can redistribute it and/or modify
// it under the terms of version 3 of the GNU Lesser General Public
// License as published by the Free Software Foundation.
//
// UserDB is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with UserDB.  If not, see <http://www.gnu.org/licenses/>.

package userdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var specialUsersURL = "http://registry.dstar.su/api/node.php"
var fixedUsersURL = "https://raw.githubusercontent.com/travisgoodspeed/md380tools/master/db/fixed.csv"
var radioidUsersURL = "https://www.radioid.net/static/users_quoted.csv"
var hamdigitalUsersURL = "https://ham-digital.org/status/users_quoted.csv"
var reflectorUsersURL = "http://registry.dstar.su/reflector.db"

var transportTimeout = 20
var clientTimeout = 300

var tr = &http.Transport{
	TLSHandshakeTimeout:   time.Duration(transportTimeout) * time.Second,
	ResponseHeaderTimeout: time.Duration(transportTimeout) * time.Second,
}

var client = &http.Client{
	Transport: tr,
	Timeout:   time.Duration(clientTimeout) * time.Second,
}

type User struct {
	ID       string
	Callsign string
	Name     string
	City     string
	State    string
	Country  string
}

type UsersDB struct {
	filename          string
	userFunc          func(*User) string
	progressCallback  func(progressCounter int) bool
	progressFunc      func() error
	progressIncrement int
	progressCounter   int
}

func newUserDB() *UsersDB {
	db := &UsersDB{
		progressFunc: func() error { return nil },
	}

	return db
}

func (db *UsersDB) setMaxProgressCount(max int) {
	db.progressFunc = func() error { return nil }
	if db.progressCallback != nil {
		db.progressIncrement = MaxProgress / max
		db.progressCounter = 0
		db.progressFunc = func() error {
			db.progressCounter += db.progressIncrement
			curProgress := db.progressCounter
			if curProgress > MaxProgress {
				curProgress = MaxProgress
			}

			if !db.progressCallback(db.progressCounter) {
				return errors.New("")
			}

			return nil
		}
		db.progressCallback(db.progressCounter)
	}
}

func (db *UsersDB) finalProgress() {
	//fmt.Fprintf(os.Stderr, "\nprogressMax %d\n", db.progressCounter/db.progressIncrement)
	if db.progressCallback != nil {
		db.progressCallback(MaxProgress)
	}
}

const (
	MinProgress = 0
	MaxProgress = 1000000
)

func (u *User) normalize() {
	u.Callsign = normalizeString(u.Callsign)
	u.Name = normalizeString(u.Name)
	u.City = normalizeString(u.City)
	u.State = normalizeString(u.State)
	u.Country = normalizeString(u.Country)
}

func normalizeString(s string) string {
	s = asciify(s)
	s = strings.TrimSpace(s)
	s = strings.Replace(s, ",", ";", -1)

	for strings.Index(s, "  ") >= 0 {
		s = strings.Replace(s, "  ", " ", -1)
	}

	return s
}

func asciify(s string) string {
	runes := []rune(s)
	strs := make([]string, len(runes))
	for i, r := range runes {
		strs[i] = transliterations[r]
	}

	return strings.Join(strs, "")
}

func getBytes(url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}

	return ioutil.ReadAll(resp.Body)
}

func getLines(url string) ([]string, error) {
	bytes, err := getBytes(url)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(bytes), "\n")

	return lines[:len(lines)-1], nil
}

func getRadioidUsers() ([]*User, error) {
	lines, err := getLines(radioidUsersURL)
	if err != nil {
		errFmt := "error getting radioid users database: %s: %s"
		err = fmt.Errorf(errFmt, radioidUsersURL, err.Error())
		return nil, err
	}

	if len(lines) < 50000 {
		errFmt := "too few radioid users database entries: %s: %d"
		err = fmt.Errorf(errFmt, radioidUsersURL, len(lines))
		return nil, err
	}

	users := make([]*User, len(lines))
	for i, line := range lines {
		line = strings.Trim(line, `"`)
		fields := strings.Split(line, `","`)

		users[i] = &User{
			ID:       fields[0],
			Callsign: fields[1],
			Name:     fields[2],
			City:     fields[3],
			State:    fields[4],
			Country:  fields[5],
		}
	}
	return users, nil
}

func getHamdigitalUsers() ([]*User, error) {
	lines, err := getLines(hamdigitalUsersURL)
	if err != nil {
		errFmt := "error getting hamdigital users database: %s: %s"
		err = fmt.Errorf(errFmt, hamdigitalUsersURL, err.Error())
		return nil, err
	}

	if len(lines) < 50000 {
		errFmt := "too few hamdigital users database entries: %s: %d"
		err = fmt.Errorf(errFmt, hamdigitalUsersURL, len(lines))
		return nil, err
	}

	users := make([]*User, len(lines))
	for i, line := range lines {
		line = strings.Trim(line, `"`)
		fields := strings.Split(line, `","`)

		users[i] = &User{
			ID:       fields[0],
			Callsign: fields[1],
			Name:     fields[2],
			City:     fields[3],
			State:    fields[4],
			Country:  fields[5],
		}
	}
	return users, nil
}

func getFixedUsers() ([]*User, error) {
	lines, err := getLines(fixedUsersURL)
	if err != nil {
		errFmt := "error getting fixed users: %s: %s"
		err = fmt.Errorf(errFmt, fixedUsersURL, err.Error())
		return nil, err
	}

	users := make([]*User, len(lines))
	for i, line := range lines {
		fields := strings.Split(line, ",")
		users[i] = &User{
			ID:       fields[0],
			Callsign: fields[1],
		}
	}
	return users, nil
}

type special struct {
	ID      string
	Country string
	Address string
}

func getSpecialURLs() ([]string, error) {
	bytes, err := getBytes(specialUsersURL)
	if err != nil {
		return nil, err
	}

	var specials []special
	err = json.Unmarshal(bytes, &specials)

	var urls []string
	for _, s := range specials {
		url := "http://" + s.Address + "/md380tools/special_IDs.csv"
		urls = append(urls, url)
	}

	return urls, nil
}

func getSpecialUsers(url string) ([]*User, error) {
	lines, err := getLines(url)
	if err != nil {
		errFmt := "error getting special users: %s: %s"
		err = fmt.Errorf(errFmt, url, err.Error())
		return nil, nil // Ignore erros on special users
	}

	users := make([]*User, len(lines))
	for i, line := range lines {
		fields := strings.Split(line, ",")
		if len(fields) < 7 {
			continue
		}
		users[i] = &User{
			ID:       fields[0],
			Callsign: fields[1],
			Name:     fields[2],
			Country:  fields[6],
		}
	}
	return users, nil
}

func getReflectorUsers() ([]*User, error) {
	lines, err := getLines(reflectorUsersURL)
	if err != nil {
		errFmt := "error getting reflector users: %s: %s"
		err = fmt.Errorf(errFmt, reflectorUsersURL, err.Error())
		return nil, err
	}

	users := make([]*User, len(lines))
	for i, line := range lines[1:] {
		line := strings.Replace(line, "@", ",", 2)
		fields := strings.Split(line, ",")
		users[i] = &User{
			ID:       fields[0],
			Callsign: fields[1],
		}
	}
	return users, nil
}

func mergeAndSort(users []*User) ([]*User, error) {
	idMap := make(map[int]*User)
	for _, u := range users {
		if u == nil || u.ID == "" {
			continue
		}
		u.ID = strings.TrimPrefix(u.ID, "#")
		id, err := strconv.ParseUint(u.ID, 10, 24)
		if err != nil {
			return nil, err
		}
		existing := idMap[int(id)]
		if existing == nil {
			idMap[int(id)] = u
			continue
		}
		// non-empty fields in later entries replace fields in earlier
		if u.Callsign != "" {
			existing.Callsign = u.Callsign
		}
		if u.Name != "" {
			existing.Name = u.Name
		}
		if u.City != "" {
			existing.City = u.City
		}
		if u.State != "" {
			existing.State = u.State
		}
		if u.Country != "" {
			existing.Country = u.Country
		}
	}

	ids := make([]int, 0, len(idMap))
	for id := range idMap {
		ids = append(ids, id)
	}

	users = make([]*User, len(ids))
	sort.Ints(ids)
	for i, id := range ids {
		users[i] = idMap[id]
	}

	return users, nil
}

type result struct {
	index int
	users []*User
	err   error
}

func do(index int, f func() ([]*User, error), resultChan chan result) {
	var r result

	r.index = index
	r.users, r.err = f()
	resultChan <- r
}

func (db *UsersDB) Users() ([]*User, error) {
	getUsersFuncs := []func() ([]*User, error){
		getFixedUsers,
		getHamdigitalUsers,
		getRadioidUsers,
		getReflectorUsers,
	}

	specialURLs, err := getSpecialURLs()
	if err != nil {
		return nil, err
	}
	for i := range specialURLs {
		url := specialURLs[i]
		f := func() ([]*User, error) {
			return getSpecialUsers(url)
		}
		getUsersFuncs = append(getUsersFuncs, f)
	}

	var users []*User
	resultCount := len(getUsersFuncs)
	resultChan := make(chan result, resultCount)

	for i, f := range getUsersFuncs {
		go do(i, f, resultChan)
	}

	db.setMaxProgressCount(resultCount)

	results := make([]result, resultCount)
	for done := 0; done < resultCount; {
		select {
		case r := <-resultChan:
			if r.err != nil {
				return nil, r.err
			}
			results[r.index] = r
			done++

			err := db.progressFunc()
			if err != nil {
				return nil, err
			}
		}
	}
	for _, r := range results {
		users = append(users, r.users...)
	}

	users, err = mergeAndSort(users)
	if err != nil {
		return nil, err
	}

	for i := range users {
		users[i].normalize()
	}

	db.finalProgress()

	return users, nil
}

func (db *UsersDB) writeSizedUsersFile() (err error) {
	file, err := os.Create(db.filename)
	if err != nil {
		return err
	}
	defer func() {
		fErr := file.Close()
		if err == nil {
			err = fErr
		}
		return
	}()

	users, err := db.Users()
	if err != nil {
		return err
	}

	strs := make([]string, len(users))
	for i, u := range users {
		strs[i] = db.userFunc(u)
	}

	length := 0
	for _, s := range strs {
		length += len(s)
	}
	fmt.Fprintf(file, "%d\n", length)

	for _, s := range strs {
		fmt.Fprint(file, s)
	}

	return nil
}

func (db *UsersDB) writeUsersFile() (err error) {
	file, err := os.Create(db.filename)
	if err != nil {
		return err
	}
	defer func() {
		fErr := file.Close()
		if err == nil {
			err = fErr
		}
		return
	}()

	fmt.Sprintln("Radio ID,CallSign,Name,NickName,City,State,Country")

	users, err := db.Users()
	if err != nil {
		return err
	}

	for _, u := range users {
		fmt.Fprint(file, db.userFunc(u))
	}

	return nil
}

func WriteMD380ToolsFile(filename string, progress func(cur int) bool) error {
	db := newUserDB()
	db.filename = filename
	db.progressCallback = progress
	db.userFunc = func(u *User) string {
		return fmt.Sprintf("%s,%s,%s,%s,%s,,%s\n",
			u.ID, u.Callsign, u.Name, u.City, u.State, u.Country)
	}

	return db.writeSizedUsersFile()
}

func WriteMD2017File(filename string, progress func(cur int) bool) error {
	db := newUserDB()
	db.filename = filename
	db.progressCallback = progress
	db.userFunc = func(u *User) string {
		return fmt.Sprintf("%s,%s,%s,,%s,%s,%s\n",
			u.ID, u.Callsign, u.Name, u.City, u.State, u.Country)
	}

	return db.writeUsersFile()
}
