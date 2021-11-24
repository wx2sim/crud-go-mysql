package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"text/template"

	"github.com/gorilla/mux"

	jwt "github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
)

type Employee struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	City string `json:"city"`
}

var mySigningKey = []byte("benchmatrix")

func isAuthorized(endpoint func(http.ResponseWriter, *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Header["Token"] != nil {

			token, err := jwt.Parse(r.Header["Token"][0], func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("There was an error")
				}
				return mySigningKey, nil
			})

			if err != nil {
				fmt.Fprintf(w, err.Error())
			}

			if token.Valid {
				endpoint(w, r)
			}
		} else {

			fmt.Fprintf(w, "Not Authorized")
		}
	})
}

func dbConn() (db *sql.DB) {
	dbDriver := "mysql"
	dbUser := "root"
	dbPass := ""
	dbName := "goblog"
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)
	if err != nil {
		panic(err.Error())
	}
	return db
}

var tmpl = template.Must(template.ParseGlob("form/*"))

func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func Index(w http.ResponseWriter, r *http.Request) {
	db := dbConn()
	selDB, err := db.Query("SELECT * FROM Employee ORDER BY id DESC")
	if err != nil {
		respondWithJson(w, http.StatusBadRequest, map[string]string{"message": "Bad request"})
	}
	emp := Employee{}
	res := []Employee{}
	for selDB.Next() {
		var id int
		var name, city string
		err = selDB.Scan(&id, &name, &city)
		if err != nil {
			respondWithJson(w, http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		emp.Id = id
		emp.Name = name
		emp.City = city
		res = append(res, emp)
	}
	respondWithJson(w, http.StatusOK, res)
	defer db.Close()
}

func New(w http.ResponseWriter, r *http.Request) {
	employee := Employee{}
	err := json.NewDecoder(r.Body).Decode(&employee)
	if err != nil {
		if err != nil {
			respondWithJson(w, http.StatusInternalServerError, map[string]string{"message": "Internal Server Error"})
		}
	}
	db := dbConn()
	name := employee.Name
	city := employee.City
	insForm, err := db.Prepare("INSERT INTO Employee(name, city) VALUES(?,?)")
	if err != nil {
		panic(err.Error())
	}
	res, err := insForm.Exec(name, city)
	if err != nil {
		respondWithJson(w, http.StatusForbidden, map[string]string{"message": "Forbiden"})
	}
	log.Println("INSERT: Name: " + name + " | City: " + city)

	defer db.Close()
	finalRes, err := res.RowsAffected()
	respondWithJson(w, http.StatusOK, finalRes)
}

func Update(w http.ResponseWriter, r *http.Request) {
	db := dbConn()
	employee := Employee{}
	err := json.NewDecoder(r.Body).Decode(&employee)
	name := employee.Name
	city := employee.City
	id := employee.Id
	insForm, err := db.Prepare("UPDATE Employee SET name=?, city=? WHERE id=?")
	if err != nil {
		respondWithJson(w, http.StatusBadRequest, map[string]string{"message": "Bad request"})
	}
	res, err := insForm.Exec(name, city, id)
	if err != nil {
		respondWithJson(w, http.StatusForbidden, map[string]string{"message": "Forbiden"})
	}
	log.Println("UPDATE: Name: " + name + " | City: " + city)
	defer db.Close()
	finalRes, err := res.RowsAffected()
	respondWithJson(w, http.StatusOK, finalRes)

}

func Delete(w http.ResponseWriter, r *http.Request) {
	db := dbConn()
	emp := r.URL.Query().Get("id")
	delForm, err := db.Prepare("DELETE FROM Employee WHERE id=?")
	if err != nil {
		respondWithJson(w, http.StatusBadRequest, map[string]string{"message": "Bad request"})
	}
	res, err := delForm.Exec(emp)
	if err != nil {
		respondWithJson(w, http.StatusForbidden, map[string]string{"message": "Forbiden"})
	}
	defer db.Close()
	finalRes, err := res.RowsAffected()
	respondWithJson(w, http.StatusOK, finalRes)

}

func HandelRequest() {
	log.Println("Server started on: http://localhost:8080")
	router := mux.NewRouter()
	router.HandleFunc("/", Index).Methods(http.MethodGet)
	router.HandleFunc("/insert", New).Methods(http.MethodPost)
	router.HandleFunc("/update/{Id}", Update).Methods(http.MethodPut)
	router.HandleFunc("/delete", Delete).Methods(http.MethodDelete)
	log.Fatal(http.ListenAndServe(":8080", router))

}

func main() {

	HandelRequest()
}
