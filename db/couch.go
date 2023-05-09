package db

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/leesper/couchdb-golang"
	"github.com/mitchellh/mapstructure"
	"strings"
	"tg_bot/models"
)

type Couch struct {
	CouchClient    *couchdb.Server
	WorkingCouchDB *couchdb.Database
}

// InitConnection make connection to DB by arguments passed
func (c *Couch) InitConnection(url, login, pass string) error {
	serv, err := couchdb.NewServer("http://" + login + ":" + pass + "@" + strings.TrimLeft(url, "http://"))
	if err != nil {
		return err
	}
	c.CouchClient = serv
	return nil
}

// CreateDatabase create database (if not created earlier)
func (c *Couch) CreateDatabase(dbName string) error {
	var err error
	if !c.CouchClient.Contains(dbName) {
		_, err = c.CouchClient.Create(dbName)
		if err != nil {
			return err
		}
	}
	existingDatabase, err := c.CouchClient.Get(dbName)
	if err != nil {
		return err
	}
	c.WorkingCouchDB = existingDatabase
	return nil
}

// SaveServiceCreds add service's creds to list of user's other creds
func (c *Couch) SaveServiceCreds(service models.Service, userName string) error {
	resp, err := c.WorkingCouchDB.Query(nil, fmt.Sprintf("username == %s", userName), nil, nil, nil, nil)
	user := &models.User{}
	// if user used bot earlier, then check his services
	if len(resp) > 0 {
		err := mapstructure.Decode(resp[0], &user)
		if err != nil {
			return err
		}
		serviceExisted := false
		for ind, oldService := range user.Services {
			// if such service already in database, then need to only update it
			if oldService.Name == service.Name {
				user.Services[ind].Login = service.Login
				user.Services[ind].Password = service.Password
				serviceExisted = true
			}
		}

		// if there is no such service for existed client, then get all user's services and add new one
		if !serviceExisted {
			user.Services = append(user.Services, service)
		}
	} else {
		newService := make([]models.Service, 0)
		newService = append(newService, service)
		user = &models.User{
			ID:       uuid.New().String(),
			Username: userName,
			Services: newService,
		}
	}
	var dataToSave map[string]interface{}
	err = mapstructure.Decode(user, &dataToSave)
	if err != nil {
		return err
	}
	_, _, err = c.WorkingCouchDB.Save(dataToSave, nil)
	if err != nil {
		return err
	}
	return nil
}

// GetService return service with creds (if so existed in db). Searching only services, that user have
func (c *Couch) GetService(serviceName string, userName string) (*models.Service, error) {
	resp, err := c.WorkingCouchDB.Query(nil, fmt.Sprintf("username == \"%s\"", userName), nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	usr := &models.User{}
	if len(resp) > 0 {
		err := mapstructure.Decode(resp[0], &usr)
		if err != nil {
			return nil, err
		}
	}
	for _, service := range usr.Services {
		if service.Name == serviceName {
			return &service, nil
		}
	}

	return nil, nil
}

// DeleteService delete service, that user have. If user doesn't have one, then error models.ErrServiceNotExistsInDB returned
func (c *Couch) DeleteService(serviceName, userName string) (bool, error) {
	resp, err := c.WorkingCouchDB.Query(nil, fmt.Sprintf("username == \"%s\"", userName), nil, nil, nil, nil)
	if err != nil {
		return false, err
	}
	usr := &models.User{}
	if len(resp) > 0 {
		if err = mapstructure.Decode(resp[0], &usr); err != nil {
			return false, err
		}
	}
	for ind, service := range usr.Services {
		// if service to delete found, then update user structure (delete one service from slice Services)
		if service.Name == serviceName {
			usr.Services = append(usr.Services[:ind], usr.Services[ind+1:]...)
			var data map[string]interface{}
			err = mapstructure.Decode(usr, &data)
			if err != nil {
				return false, err
			}
			_, _, err = c.WorkingCouchDB.Save(data, nil)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, fmt.Errorf("cannot delete service %s. %w", serviceName, models.ErrServiceNotExistsInDB)
}
