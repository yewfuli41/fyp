package graphErrs

import "github.com/vektah/gqlparser/v2/gqlerror"

type graphQLErrorHandler func(error) *gqlerror.Error

var handlers []graphQLErrorHandler

func registerGraphQLErrorHandler(handler graphQLErrorHandler) {
	handlers = append(handlers, handler)
}

func ToGraphQLError(err error) error {
	for _, handler := range handlers {
		if gqlErr := handler(err); gqlErr != nil {
			return gqlErr
		}
	}

	return err
}
