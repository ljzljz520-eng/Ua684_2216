package httpapi

import (
	"net/url"
	"strings"

	"service-request-dispatch/internal/model"
)

type RouteInfo struct {
	Resource string
	ID       string
	Action   string
}

func ParseRoute(path string) RouteInfo {
	clean := strings.Trim(path, "/")
	parts := strings.Split(clean, "/")
	if len(parts) < 2 || parts[0] != "api" {
		return RouteInfo{}
	}
	result := RouteInfo{Resource: parts[1]}
	if len(parts) > 2 {
		result.ID, _ = url.PathUnescape(parts[2])
	}
	if len(parts) > 3 {
		result.Action, _ = url.PathUnescape(parts[3])
	}
	return result
}

func FilterFromValues(values url.Values) model.RequestFilter {
	return model.RequestFilter{Status: model.RequestStatus(values.Get("status")), GroupID: values.Get("group"), Assignee: values.Get("assignee"), Tag: values.Get("tag")}
}

func IsReadMethod(method string) bool { return method == "GET" || method == "HEAD" }

func IsMutationMethod(method string) bool {
	return method == "POST" || method == "PATCH" || method == "DELETE"
}
