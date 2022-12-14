package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/xeipuuv/gojsonschema"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func db() *mongo.Client {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to MongoDB!")
	return client
}

type Response struct {
	Msg    string      `json:"msg"`
	Data   interface{} `json:"data"`
	Status int         `json:"status"`
}

type Response2 struct {
	Msg        string      `json:"msg"`
	Data       interface{} `json:"data"`
	Status     int         `json:"status"`
	TotalCount int64       `json:"totalcount"`
}

var userCollection = db().Database("chandan").Collection("student")

func validate(schemaPath string, payload map[string]interface{}) (interface{}, error) {
	payload_bytes, _ := json.Marshal(payload)
	schema_bytes, _ := ioutil.ReadFile(schemaPath)
	schemaLoader := gojsonschema.NewBytesLoader(schema_bytes)
	documentLoader := gojsonschema.NewBytesLoader(payload_bytes)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return err, errors.New("internal server error")
	}
	if result.Valid() {
		return nil, nil
	} else {

		validationErrors := make([]string, 0)
		for _, desc := range result.Errors() {
			validationErrors = append(validationErrors, desc.String())
		}
		return validationErrors, errors.New("invalid payload")
	}
}

func ResponseHandler(r string, d interface{}, status int, w http.ResponseWriter) {
	w.WriteHeader(status)
	response := Response{Msg: r, Data: d, Status: status}
	json.NewEncoder(w).Encode(response)
}

func ResponseHandler2(r string, d interface{}, status int, totalcount int64, w http.ResponseWriter) {
	w.WriteHeader(status)
	response2 := Response2{Msg: r, Data: d, Status: status, TotalCount: totalcount}
	json.NewEncoder(w).Encode(response2)
}

func createProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var person = make(map[string]interface{})
	json.NewDecoder(r.Body).Decode(&person)
	cpath := "C:/Users/avita/Desktop/GO/mongodb2/createschema.json"
	result, err := validate(cpath, person)
	if err != nil {
		fmt.Println(err)
		ResponseHandler("validation error", result, 400, w)
		return
	}
	err = userCollection.FindOne(context.TODO(), bson.M{"name": person["name"]}).Decode(&person)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			fmt.Println(err)
			ResponseHandler("internal server error", nil, 500, w)
			return
		}
	} else {
		ResponseHandler("user already exists", nil, 409, w)
		return
	}
	delete(person, "_id")

	insertResult, err := userCollection.InsertOne(context.TODO(), person)
	if err != nil {
		fmt.Println(err)
		ResponseHandler("Internal Server Error", nil, 500, w)
		return
	}
	err = userCollection.FindOne(context.TODO(), bson.M{"_id": insertResult.InsertedID}).Decode(&person)
	if err != nil {
		fmt.Println(err)
		ResponseHandler("Internal Server Error", nil, 500, w)
		return
	}
	ResponseHandler("User Inserted Successfully", person, 201, w)

}

func getUserProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	id, ok := vars["id"]
	var person = make(map[string]interface{})
	if !ok {
		fmt.Println("id is missing in params")
		ResponseHandler("id missing in params", nil, 400, w)
		return
	}
	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		fmt.Println(err)
		ResponseHandler("invalid id", nil, 400, w)
		return
	}
	err = userCollection.FindOne(context.TODO(), bson.M{"_id": _id}).Decode(&person)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println(err)
			ResponseHandler("user not found", nil, 404, w)
			return
		}
		fmt.Println(err)
		ResponseHandler("internal server error", nil, 500, w)
		return

	}
	ResponseHandler("user fetched successfully", person, 200, w)
}

func updateProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var person = make(map[string]interface{})
	json.NewDecoder(r.Body).Decode(&person)
	upath := "C:/Users/avita/Desktop/GO/mongodb2/updateschema.json"
	result, err := validate(upath, person)
	if err != nil && result != nil {
		fmt.Println(err)
		ResponseHandler("invalid payload", result, 400, w)
		return
	}

	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		fmt.Println("id is missing in params")
		ResponseHandler("id missing in params", nil, 400, w)
		return
	}
	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		fmt.Println(err)
		ResponseHandler("invalid payload", nil, 400, w)
		return
	}

	existinguser := make(map[string]interface{})
	err = userCollection.FindOne(context.TODO(), bson.M{"_id": _id}).Decode(&existinguser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println(err)
			ResponseHandler("user not found", nil, 404, w)
		} else {
			ResponseHandler("internal server error", nil, 500, w)
		}
		return
	}
	pname, pok := person["name"].(string)
	if pok {
		if pname != existinguser["name"].(string) {
			err = userCollection.FindOne(context.TODO(), bson.M{"name": pname}).Decode(&existinguser)
			if err != nil {
				if err != mongo.ErrNoDocuments {
					fmt.Println(err)
					ResponseHandler("internal server error", nil, 500, w)
					return
				}
			} else {
				fmt.Println("user already exists")
				ResponseHandler("user already exists", nil, 409, w)
				return
			}
		}
	}
	delete(person, "_id")
	_, err = userCollection.UpdateOne(context.TODO(), bson.M{"_id": _id}, bson.M{"$set": person})
	if err != nil {
		fmt.Println(err)
		ResponseHandler("Internal Server Error", nil, 500, w)
		return
	}
	ResponseHandler("User updated successfully", nil, 200, w)
}

func deleteProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		fmt.Println("id is missing in params")
		ResponseHandler("Id missing in params", nil, 400, w)
		return
	}
	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		fmt.Println(err)
		ResponseHandler("Invalid id", nil, 400, w)
		return
	}
	var person = make(map[string]interface{})

	err = userCollection.FindOneAndDelete(context.TODO(), bson.M{"_id": _id}).Decode(&person)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println(err)
			ResponseHandler("User not found", nil, 404, w)
		} else {
			ResponseHandler("internal server error", nil, 500, w)
		}
		return
	}
	ResponseHandler("User deleted successfully", nil, 200, w)
}

func getAllUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	p := make(map[string]interface{})
	filters := r.FormValue("filters")
	page := r.FormValue("page")
	limit := r.FormValue("size")
	err := json.Unmarshal([]byte(filters), &p)
	if err != nil {
		var result []primitive.M
		dc, err := userCollection.CountDocuments(context.TODO(), bson.M{})
		if err != nil {
			if err == mongo.ErrNoDocuments {
				fmt.Println(err)
				ResponseHandler("DB is empty", nil, 500, w)
				return
			}
			fmt.Println(err)
			ResponseHandler("Internal Server Error", nil, 500, w)
			return
		}
		cur, err := userCollection.Find(context.TODO(), bson.M{})
		if err != nil {
			if err == mongo.ErrNoDocuments {
				fmt.Println(err)
				ResponseHandler("DB is empty", nil, 500, w)
				return
			}
			fmt.Println(err)
			ResponseHandler("Internal Server Error", nil, 500, w)
			return
		}
		for cur.Next(context.TODO()) { //Next() gets the next document for corresponding cursor
			var elem primitive.M
			err := cur.Decode(&elem)
			if err != nil {
				fmt.Println(err)
				ResponseHandler("Internal Server Error", nil, 500, w)
				return
			}
			result = append(result, elem) // appending document pointed by Next()
		}
		cur.Close(context.TODO()) // close the cursor once stream of documents has exhausted
		ResponseHandler2("Users fecthed successfully", result, 200, dc, w)
		return
	}
	if page == "" {
		page = "1"
	}
	if limit == "" {
		limit = "10"
	}
	var results []primitive.M
	pag, err := strconv.ParseInt(page, 10, 64)
	if err != nil {
		fmt.Println(err)
		ResponseHandler("Invalid page", nil, 400, w)
		return
	}
	lim, err := strconv.ParseInt(limit, 10, 64)
	if err != nil {
		fmt.Println(err)
		ResponseHandler("Invalid page", nil, 400, w)
		return
	}
	skip := (pag - 1) * lim
	opts := options.FindOptions{Skip: &skip, Limit: &lim}
	c, err := userCollection.CountDocuments(context.TODO(), p)
	if err != nil {
		fmt.Println(err)
		ResponseHandler("Internal Server Error", nil, 500, w)
		return
	}
	cur, err := userCollection.Find(context.TODO(), p, &opts)
	if err != nil {
		fmt.Println(err)
		ResponseHandler("Internal Server Error", nil, 500, w)
		return
	}
	for cur.Next(context.TODO()) { //Next() gets the next document for corresponding cursor
		var elem primitive.M
		err := cur.Decode(&elem)
		if err != nil {
			fmt.Println(err)
			ResponseHandler("Internal Server Error", nil, 500, w)
			return
		}
		results = append(results, elem) // appending document pointed by Next()
	}
	cur.Close(context.TODO()) // close the cursor once stream of documents has exhausted
	ResponseHandler2("Users fecthed successfully", results, 200, c, w)
}

func main() {

	route := mux.NewRouter()
	s := route.PathPrefix("/api").Subrouter() //Base Path

	//Routes

	s.HandleFunc("/createProfile", createProfile).Methods("POST")
	s.HandleFunc("/getAllUsers", getAllUsers).Methods("GET")
	s.HandleFunc("/getUserProfile/{id}", getUserProfile).Methods("GET")
	s.HandleFunc("/updateProfile/{id}", updateProfile).Methods("PUT")
	s.HandleFunc("/deleteProfile/{id}", deleteProfile).Methods("DELETE")

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
	})
	handler := c.Handler(route)

	log.Fatal(http.ListenAndServe(":6000", handler))
}
