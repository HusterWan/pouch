package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/alibaba/pouch/apis/types"
	"github.com/alibaba/pouch/pkg/httputils"
	"github.com/alibaba/pouch/pkg/utils"
)

func (s *Server) ping(context context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte{'O', 'K'})
	return
}

func (s *Server) info(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	info, err := s.SystemMgr.Info()
	if err != nil {
		return err
	}

	if utils.IsStale(ctx, req) {
		ba, err := json.Marshal(info)
		if err != nil {
			return err
		}
		m := make(map[string]interface{})
		err = json.Unmarshal(ba, &m)
		if err != nil {
			return err
		}
		rootDir := m["PouchRootDir"]
		delete(m, "PouchRootDir")
		m["DockerRootDir"] = rootDir
		return EncodeResponse(rw, http.StatusOK, m)
	}

	return EncodeResponse(rw, http.StatusOK, info)
}

func (s *Server) version(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	version, err := s.SystemMgr.Version()
	if err != nil {
		return err
	}
	if utils.IsStale(ctx, req) {
		version.Version = "1.12.6"
	}
	return EncodeResponse(rw, http.StatusOK, version)
}

func (s *Server) updateDaemon(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	cfg := &types.DaemonUpdateConfig{}
	if err := json.NewDecoder(req.Body).Decode(cfg); err != nil {
		return httputils.NewHTTPError(err, http.StatusBadRequest)
	}

	// TODO: validate cfg in details

	return s.SystemMgr.UpdateDaemon(cfg)
}

func (s *Server) auth(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	auth := types.AuthConfig{}
	if err := json.NewDecoder(req.Body).Decode(&auth); err != nil {
		return httputils.NewHTTPError(err, http.StatusBadRequest)
	}

	token, err := s.SystemMgr.Auth(&auth)
	if err != nil {
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	authResp := types.AuthResponse{
		Status:        "Login Succeeded",
		IdentityToken: token,
	}
	return EncodeResponse(rw, http.StatusOK, authResp)
}
