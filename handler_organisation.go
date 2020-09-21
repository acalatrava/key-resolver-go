package main

import (
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/bitmaelum/bitmaelum-suite/pkg/bmcrypto"
	"github.com/bitmaelum/bitmaelum-suite/pkg/proofofwork"
	"github.com/bitmaelum/key-resolver-go/organisation"
	"log"
)

type organisationUploadBody struct {
	PublicKey bmcrypto.PubKey         `json:"public_key"`
	Proof     proofofwork.ProofOfWork `json:"proof"`
}

func getOrganisationHash(hash string, _ events.APIGatewayV2HTTPRequest) *events.APIGatewayV2HTTPResponse {
	repo := organisation.GetResolveRepository()
	info, err := repo.Get(hash)
	if err != nil && err != organisation.ErrNotFound {
		log.Print(err)
		return createError("hash not found", 404)
	}

	if info == nil {
		log.Print(err)
		return createError("hash not found", 404)
	}

	data := jsonOut{
		"hash":       info.Hash,
		"public_key": info.PubKey,
	}

	return createOutput(data, 200)
}

func postOrganisationHash(hash string, req events.APIGatewayV2HTTPRequest) *events.APIGatewayV2HTTPResponse {
	repo := organisation.GetResolveRepository()
	current, err := repo.Get(hash)
	if err != nil && err != organisation.ErrNotFound {
		log.Print(err)
		return createError("error while posting record", 500)
	}

	uploadBody := &organisationUploadBody{}
	err = json.Unmarshal([]byte(req.Body), uploadBody)
	if err != nil {
		log.Print(err)
		return createError("invalid data", 400)
	}

	if !validateOrganisationBody(*uploadBody) {
		return createError("invalid data", 400)
	}

	if current == nil {
		// Does not exist yet
		return createOrganisation(hash, *uploadBody)
	}

	// Try update
	return updateOrganisation(*uploadBody, req, current)
}

func deleteOrganisationHash(hash string, req events.APIGatewayV2HTTPRequest) *events.APIGatewayV2HTTPResponse {
	repo := organisation.GetResolveRepository()
	current, err := repo.Get(hash)
	if err != nil {
		log.Print(err)
		return createError("error while fetching record", 500)
	}

	if current == nil {
		return createError("cannot find record", 404)
	}

	if !validateSignature(req, current.PubKey, current.Hash) {
		return createError("unauthenticated", 401)
	}

	res, err := repo.Delete(current.Hash)
	if err != nil || res == false {
		log.Print(err)
		return createError("error while deleting record", 500)
	}

	return createOutput("ok", 200)
}

func updateOrganisation(uploadBody organisationUploadBody, req events.APIGatewayV2HTTPRequest, current *organisation.ResolveInfoType) *events.APIGatewayV2HTTPResponse {
	if !validateSignature(req, current.PubKey, current.Hash) {
		return createError("unauthenticated", 401)
	}

	repo := organisation.GetResolveRepository()
	res, err := repo.Update(current, uploadBody.PublicKey.String(), uploadBody.Proof.String())

	if err != nil || res == false {
		log.Print(err)
		return createError("error while updating: ", 500)
	}

	return createOutput("updated", 200)
}

func createOrganisation(hash string, uploadBody organisationUploadBody) *events.APIGatewayV2HTTPResponse {
	if !uploadBody.Proof.IsValid() {
		return createError("incorrect proof-of-work", 401)
	}

	repo := organisation.GetResolveRepository()
	res, err := repo.Create(hash, uploadBody.PublicKey.String(), uploadBody.Proof.String())

	if err != nil || res == false {
		log.Print(err)
		return createError("error while creating: ", 500)
	}

	return createOutput("created", 201)
}

func validateOrganisationBody(body organisationUploadBody) bool {
	// PubKey and proof are already validated through the JSON marshalling
	return true
}