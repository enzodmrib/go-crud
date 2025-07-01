package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"rocketseat/models"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

func NewHandler(db models.DB[*models.User]) http.Handler {
	r := chi.NewMux()

	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)

	r.Get("/users", handleFindAll(db))
	r.Get("/users/{id}", handleFindById(db))
	r.Post("/users", handleInsert(db))
	r.Put("/users/{id}", handleUpdate(db))
	r.Delete("/users/{id}", handleDelete(db))

	return r
}

type UserResponse struct {
	ID uuid.UUID `json:"id"`
	*models.User
}

func validateRequestBodyFields(body io.ReadCloser, schemaObj any) (*models.User, error) {
	// Why not build a function that gets generic schemas and checks if the request body follows them?

	var user models.User
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&user); err != nil {
		return nil, err
	}

	schemaObjJson, err := json.Marshal(schemaObj)
	if err != nil {
		return nil, err
	}
	var schemaObjMap map[string]interface{}
	if err := json.Unmarshal(schemaObjJson, &schemaObjMap); err != nil {
		return nil, err
	}

	userJson, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}
	var userMap map[string]interface{}
	if err := json.Unmarshal(userJson, &userMap); err != nil {
		return nil, err
	}

	for key := range schemaObjMap {
		if userMap[key] == nil {
			return nil, errors.New("please provide FirstName LastName and bio for the user")
		}
	}

	return &user, nil
}

func handleFindAll(db models.DB[*models.User]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var result []UserResponse

		for key, value := range db {
			result = append(result, UserResponse{ID: key, User: value})
		}

		jsonResult, err := json.Marshal(result)
		if err != nil {
			http.Error(w, "Error parsing response", http.StatusInternalServerError)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResult)
	}
}

func handleFindById(db models.DB[*models.User]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		parsedID, err := uuid.Parse(id)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
		}

		user, ok := db[parsedID]
		if !ok {
			http.Error(w, "User not found", http.StatusNotFound)
		}

		userResponse := UserResponse{ID: parsedID, User: user}

		jsonUser, err := json.Marshal(userResponse)
		if err != nil {
			http.Error(w, "Error parsing response", http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(jsonUser)
	}
}

func handleInsert(db models.DB[*models.User]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var userModel models.User
		user, err := validateRequestBodyFields(r.Body, userModel)
		if err != nil {
			slog.Error("Request body validation error", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
		}

		userId := uuid.New()

		db[userId] = user

		userResponse := UserResponse{ID: userId, User: user}

		jsonUser, err := json.Marshal(userResponse)
		if err != nil {
			http.Error(w, "Error while parsing the response", http.StatusInternalServerError)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write(jsonUser)
	}
}

func handleUpdate(db models.DB[*models.User]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		id := chi.URLParam(r, "id")
		parsedID, err := uuid.Parse(id)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
		}

		var userModel models.User
		user, err := validateRequestBodyFields(r.Body, userModel)
		if err != nil {
			slog.Error("Request body validation error", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
		}

		_, ok := db[parsedID]
		if !ok {
			http.Error(w, "User not found", http.StatusNotFound)
		}

		db[parsedID] = user

		userResponse := UserResponse{ID: parsedID, User: user}

		jsonUser, err := json.Marshal(userResponse)
		if err != nil {
			http.Error(w, "Error while parsing the response", http.StatusInternalServerError)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(jsonUser)
	}
}
func handleDelete(db models.DB[*models.User]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		id := chi.URLParam(r, "id")

		parsedID, err := uuid.Parse(id)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
		}

		_, ok := db[parsedID]

		if !ok {
			http.Error(w, "User not found", http.StatusNotFound)
		}

		delete(db, parsedID)

		w.WriteHeader(http.StatusNoContent)
	}
}

func handleInsert_EXPERIMENTAL(db models.DB[*models.User]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// there are two common ways to read a request.

		// =======================================================

		// first is using the io.ReadAll, which reads the http request stream entirely
		// however, there is a catch - if the user sends a body too large, it could overload system memory, meaning its prone to attacks
		// but there are safe ways to do so

		// this way, we limit reading on the body by a determined ammount of bytes
		maxBytes := int64(1024 * 1024) // 1 MB limit
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		// IMPORTANT: Once the body stream is read, it's consumed and cannot be read again.
		bodyBytes, err := io.ReadAll(r.Body)

		if err != nil {
			slog.Error("error reading request body", "error", err)
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}

		var payload models.User
		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			slog.Error("error unmarshaling request body to payload", "error", err)
			http.Error(w, "Error unmarshaling request body to payload", http.StatusBadRequest)
			return
		}

		defer r.Body.Close()

		// reseting the body is necessary to read it again
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		// =======================================================

		// second is using json.NewDecoder, which simply decodes the body stream into the struct.
		// this way is more straightforward and safe

		var user models.User
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&user); err != nil {
			slog.Error("error decoding request body to user", "error", err)
			http.Error(w, "Error unmarshaling request body to payload", http.StatusBadRequest)
			return
		}

		// =======================================================

		// from now on, things are handled equally for both methods

		w.Header().Set("Content-Type", "application/json")
		data, err := json.Marshal(user)
		if err != nil {
			slog.Error("error marshaling request body to payload", "error", err)
			return
		}
		w.Write(data)
	}
}
