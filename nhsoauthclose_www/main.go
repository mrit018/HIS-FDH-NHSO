// main.go
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	_"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Visit represents a medical visit record
type Visit struct {
	Vstdate        string `json:"vstdate"`
	Vsttime        string `json:"vsttime"`
	CID            string `json:"cid"`
	HN             string `json:"hn"`
	Birthday       string `json:"birthday"`
	VN             string `json:"vn"`
	Ptname         string `json:"ptname"`
	CodePttype     string `json:"code_pttype"`
	NamePttype     string `json:"name_pttype"`
	SpcltyName     string `json:"spclty_name"`
	DepartmentName string `json:"department_name"`
	TotalAmount    string `json:"totalAmount"`
	AuthCode       string `json:"auth_code"`
	Telephone      string `json:"telephone"`
}

// FilterCriteria represents the filter options from the frontend
type FilterCriteria struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Pttype    string `json:"pttype"`
	Spclty    string `json:"spclty"`
	HN        string `json:"hn"`
	VN        string `json:"vn"`
}

// DB connection global variable
var db *sql.DB

func initDB() {
	var err error
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"))

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Successfully connected to PostgreSQL")
}

func getVisitsHandler(w http.ResponseWriter, r *http.Request) {
	var criteria FilterCriteria

	// Parse query parameters
	queryParams := r.URL.Query()
	criteria.StartDate = queryParams.Get("start_date")
	criteria.EndDate = queryParams.Get("end_date")
	criteria.Pttype = queryParams.Get("pttype")
	criteria.Spclty = queryParams.Get("spclty")
	criteria.HN = queryParams.Get("hn")
	criteria.VN = queryParams.Get("vn")

	// Build WHERE clause conditions
	var conditions []string
	var args []interface{}
	argCount := 1

	// Date filter
	if criteria.StartDate != "" && criteria.EndDate != "" {
		conditions = append(conditions, fmt.Sprintf("o.vstdate BETWEEN $%d AND $%d", argCount, argCount+1))
		args = append(args, criteria.StartDate, criteria.EndDate)
		argCount += 2
	} else if criteria.StartDate != "" {
		conditions = append(conditions, fmt.Sprintf("o.vstdate >= $%d", argCount))
		args = append(args, criteria.StartDate)
		argCount++
	} else if criteria.EndDate != "" {
		conditions = append(conditions, fmt.Sprintf("o.vstdate <= $%d", argCount))
		args = append(args, criteria.EndDate)
		argCount++
	}

	// PTType filter
	if criteria.Pttype != "" {
		conditions = append(conditions, fmt.Sprintf("o.pttype = $%d", argCount))
		args = append(args, criteria.Pttype)
		argCount++
	}

	// Specialty filter
	if criteria.Spclty != "" {
		conditions = append(conditions, fmt.Sprintf("o.spclty = $%d", argCount))
		args = append(args, criteria.Spclty)
		argCount++
	}

	// HN filter
	if criteria.HN != "" {
		conditions = append(conditions, fmt.Sprintf("o.hn = $%d", argCount))
		args = append(args, criteria.HN)
		argCount++
	}

	// VN filter
	if criteria.VN != "" {
		conditions = append(conditions, fmt.Sprintf("o.vn = $%d", argCount))
		args = append(args, criteria.VN)
		argCount++
	}

	// Build the WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ") + " AND "
	} else {
		whereClause = "WHERE "
	}

	// Fixed conditions
	whereClause += `
		P.nationality IN ('99') 
		AND P.citizenship IN ('99') 
		AND o.cur_dep IN ('999') 
		AND P.CID NOT LIKE '0%'
		AND v.item_money != 0
		AND vpt.auth_code NOT LIKE 'EP%'
	`

	// Build the final query with telephone field
	query := fmt.Sprintf(`
		SELECT
			o.vstdate,
			o.vsttime,
			P.CID,
			o.hn,
			P.birthday,
			o.vn,
			CAST(CONCAT(P.pname, P.fname, ' ', P.lname) AS CHAR(250)) AS ptname,
			T.hipdata_code  AS code_pttype,
			T.NAME AS name_pttype,
			s.NAME AS spclty_name,
			K.department AS department_name,
			v.item_money AS totalAmount,
			vpt.auth_code,
			CASE 
				WHEN REGEXP_REPLACE(COALESCE(P.mobile_phone_number, ''), '[^0-9]', '', 'g') != '' 
					THEN REGEXP_REPLACE(P.mobile_phone_number, '[^0-9]', '', 'g') 
				WHEN REGEXP_REPLACE(COALESCE(P.hometel, ''), '[^0-9]', '', 'g') != '' 
					THEN REGEXP_REPLACE(P.hometel, '[^0-9]', '', 'g')
				WHEN REGEXP_REPLACE(COALESCE(P.informtel, ''), '[^0-9]', '', 'g') != '' 
					THEN REGEXP_REPLACE(P.informtel, '[^0-9]', '', 'g')
				ELSE '0'
			END AS telephone
		FROM
			ovst o
			LEFT OUTER JOIN vn_stat v ON v.vn = o.vn
			LEFT OUTER JOIN opdscreen oc ON oc.vn = o.vn
			LEFT OUTER JOIN patient P ON P.hn = o.hn
			LEFT OUTER JOIN pttype T ON T.pttype = o.pttype
			LEFT OUTER JOIN spclty s ON s.spclty = o.spclty
			LEFT OUTER JOIN kskdepartment K ON K.depcode = o.cur_dep
			LEFT OUTER JOIN visit_pttype vpt ON vpt.vn = o.vn 
				AND vpt.pttype = o.pttype
		%s
		ORDER BY
			o.vsttime DESC
	`, whereClause)

	// Execute query
	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var visits []Visit
	for rows.Next() {
		var v Visit
		err := rows.Scan(
			&v.Vstdate, &v.Vsttime, &v.CID, &v.HN, &v.Birthday, &v.VN,
			&v.Ptname, &v.CodePttype, &v.NamePttype, &v.SpcltyName,
			&v.DepartmentName, &v.TotalAmount, &v.AuthCode, &v.Telephone,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		visits = append(visits, v)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(visits)
}

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize database connection
	initDB()
	defer db.Close()

	// Create router
	router := mux.NewRouter()

	// Define routes
	router.HandleFunc("/api/visits", enableCORS(getVisitsHandler)).Methods("GET", "OPTIONS")

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
