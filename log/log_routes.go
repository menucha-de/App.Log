package log

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/peramic/utils"
)

//NewRouter router constructor
func NewRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		var handler http.Handler
		handler = route.HandlerFunc

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
}

//LogRoutes routes
var routes = utils.Routes{

	utils.Route{
		Name:        "GetLogLevels",
		Method:      "GET",
		Pattern:     "/rest/log/levels",
		HandlerFunc: getLogLevels,
	},
	utils.Route{
		Name:        "GetLogTargets",
		Method:      "GET",
		Pattern:     "/rest/log/targets/{host}",
		HandlerFunc: getLogTargets,
	},

	utils.Route{
		Name:        "GetLogSize",
		Method:      "GET",
		Pattern:     "/rest/log/{host}/{target}/{level}",
		HandlerFunc: getLogSize,
	},
	utils.Route{
		Name:        "GetLogEntries",
		Method:      "GET",
		Pattern:     "/rest/log/{host}/{target}/{level}/{limit}/{offset}/{order}",
		HandlerFunc: getLogEntries,
	},
	utils.Route{
		Name:        "DeleteLogEntries",
		Method:      "DELETE",
		Pattern:     "/rest/log/{host}/{target}",
		HandlerFunc: deleteLogEntries,
	},
	utils.Route{
		Name:        "SetLogLevel",
		Method:      "PUT",
		Pattern:     "/rest/log/{host}/{target}",
		HandlerFunc: setLogLevel,
	},

	utils.Route{
		Name:        "GetLogFile",
		Method:      "GET",
		Pattern:     "/rest/log/{host}/{target}/{level}/export",
		HandlerFunc: getLogFile,
	},
	utils.Route{
		Name:        "GetLogHosts",
		Method:      "GET",
		Pattern:     "/rest/log/hosts",
		HandlerFunc: getLogHosts,
	},
}
