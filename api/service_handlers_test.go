package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsuru/rpaas-operator/pkg/apis/extensions/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func setupTest(t *testing.T) {
	scheme, err := v1alpha1.SchemeBuilder.Build()
	require.Nil(t, err)
	cli = fake.NewFakeClientWithScheme(scheme)

	err = cli.Create(context.TODO(), &v1alpha1.RpaasPlan{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RpaasPlan",
			APIVersion: "extensions.tsuru.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myplan",
			Namespace: NAMESPACE,
		},
	})
	require.Nil(t, err)
	err = cli.Create(context.TODO(), &v1alpha1.RpaasInstance{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RpaasInstance",
			APIVersion: "extensions.tsuru.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "firstinstance",
			Namespace: NAMESPACE,
		},
	})
	require.Nil(t, err)
}

func Test_serviceCreate(t *testing.T) {
	setupTest(t)

	testCases := []struct {
		requestBody  string
		expectedCode int
		expectedBody string
	}{
		{
			"",
			http.StatusBadRequest,
			"name is required",
		},
		{
			"name=rpaas",
			http.StatusBadRequest,
			"plan is required",
		},
		{
			"name=rpaas&plan=myplan",
			http.StatusBadRequest,
			"team name is required",
		},
		{
			"name=rpaas&plan=plan2&team=myteam",
			http.StatusBadRequest,
			"invalid plan",
		},
		{
			"name=firstinstance&plan=myplan&team=myteam",
			http.StatusConflict,
			"firstinstance instance already exists",
		},
		{
			"name=otherinstance&plan=myplan&team=myteam",
			http.StatusCreated,
			"",
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("when body == %q", testCase.requestBody), func(t *testing.T) {
			e := echo.New()
			request := httptest.NewRequest(http.MethodPost, "/resources", strings.NewReader(testCase.requestBody))
			request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			recorder := httptest.NewRecorder()
			context := e.NewContext(request, recorder)
			err := serviceCreate(context)
			assert.Nil(t, err)
			e.HTTPErrorHandler(err, context)
			assert.Equal(t, testCase.expectedCode, recorder.Code)
			assert.Equal(t, testCase.expectedBody, recorder.Body.String())
		})
	}
}

func Test_serviceDelete(t *testing.T) {
	setupTest(t)

	testCases := []struct {
		instanceName string
		expectedCode int
		expectedBody string
	}{
		{
			"",
			http.StatusBadRequest,
			"name is required",
		},
		{
			"unknown",
			http.StatusNotFound,
			"",
		},
		{
			"firstinstance",
			http.StatusOK,
			"",
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("when instance name == %q", testCase.instanceName), func(t *testing.T) {
			e := echo.New()
			request := httptest.NewRequest(http.MethodDelete, "/resources/"+testCase.instanceName, nil)
			recorder := httptest.NewRecorder()
			context := e.NewContext(request, recorder)
			context.SetParamNames("instance")
			context.SetParamValues(testCase.instanceName)
			err := serviceDelete(context)
			assert.Nil(t, err)
			e.HTTPErrorHandler(err, context)
			assert.Equal(t, testCase.expectedCode, recorder.Code)
			assert.Equal(t, testCase.expectedBody, recorder.Body.String())
		})
	}
}

func Test_servicePlans(t *testing.T) {
	setupTest(t)

	e := echo.New()
	request := httptest.NewRequest(http.MethodGet, "/resources/plans", nil)
	recorder := httptest.NewRecorder()
	context := e.NewContext(request, recorder)
	err := servicePlans(context)
	assert.Nil(t, err)
	e.HTTPErrorHandler(err, context)
	assert.Equal(t, http.StatusOK, recorder.Code)

	type result struct {
		Name, Description string
	}
	r := []result{}
	err = json.Unmarshal(recorder.Body.Bytes(), &r)
	require.Nil(t, err)
	expected := []result{{Name: "myplan", Description: "no plan description"}}
	assert.Equal(t, expected, r)
}
