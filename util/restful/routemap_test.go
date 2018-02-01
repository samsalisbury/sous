package restful

import "testing"

func TestRouteMap_URIFor_success(t *testing.T) {

	makeTestRoutes := func() RouteMap {
		return RouteMap{
			{
				Name: "root",
				Path: "/",
			},
			{
				Name: "simple",
				Path: "/simple",
			},
			{
				Name: "endparam",
				Path: "/simple/:endparam",
			},
			{
				Name: "midparam_blah",
				Path: "/simple/:midparam/blah",
			},
			{
				Name: "twoparam",
				Path: "/twoparam/:param1/:param2",
			},
			{
				Name: "twoparam_blah",
				Path: "/twoparam/:param1/:param2/blah",
			},
			{
				Name: "gappedparams",
				Path: "/gapped/:param1/gap/:param2",
			},
			{
				Name: "gappedparams_blah",
				Path: "/gapped/:param1/gap/:param2/blah",
			},
		}
	}

	testCases := []struct {
		name       string
		pathParams map[string]string
		wantURI    string
	}{
		{
			name:    "root",
			wantURI: "/",
		},
		{
			name:    "simple",
			wantURI: "/simple",
		},
		{
			name:       "endparam",
			pathParams: map[string]string{"endparam": "one"},
			wantURI:    "/simple/one",
		},
		{
			name:       "endparam",
			pathParams: map[string]string{"endparam": "two"},
			wantURI:    "/simple/two",
		},
		{
			name:       "midparam_blah",
			pathParams: map[string]string{"midparam": "one"},
			wantURI:    "/simple/one/blah",
		},
		{
			name:       "midparam_blah",
			pathParams: map[string]string{"midparam": "two"},
			wantURI:    "/simple/two/blah",
		},
		{
			name:       "twoparam",
			pathParams: map[string]string{"param1": "one", "param2": "two"},
			wantURI:    "/twoparam/one/two",
		},
		{
			name:       "twoparam_blah",
			pathParams: map[string]string{"param1": "one", "param2": "two"},
			wantURI:    "/twoparam/one/two/blah",
		},
		{
			name:       "gappedparams",
			pathParams: map[string]string{"param1": "one", "param2": "two"},
			wantURI:    "/gapped/one/gap/two",
		},
		{
			name:       "gappedparams_blah",
			pathParams: map[string]string{"param1": "one", "param2": "two"},
			wantURI:    "/gapped/one/gap/two/blah",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"_"+tc.wantURI, func(t *testing.T) {
			routes := makeTestRoutes()
			gotURI, err := routes.URIFor(tc.name, tc.pathParams)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if gotURI != tc.wantURI {
				t.Errorf("got URI %q; want %q", gotURI, tc.wantURI)
			}
		})
	}

}
