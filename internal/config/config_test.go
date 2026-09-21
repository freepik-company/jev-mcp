package config

import "testing"

func TestConfiguration(t *testing.T) {
	for _, base := range []string{"file:///tmp/key", "http://example.com", "https://user:password@host", "https://host?key=secret", "https://host#fragment", "://"} {
		if _, err := Load(func(name string) string { return map[string]string{"BASE_URL": base, "API_KEY": "key"}[name] }); err == nil {
			t.Errorf("aceptó BASE_URL inválida: %s", base)
		}
	}
	for _, key := range []string{"", " ", "${secrets:MISSING}"} {
		if _, err := Load(func(name string) string {
			if name == "API_KEY" {
				return key
			}
			return ""
		}); err == nil {
			t.Error("aceptó una credencial ausente o sin resolver")
		}
	}
	for base, endpoint := range map[string]string{
		"":                                  "https://openrouter.ai/api",
		"https://api.typesafe.ai":           "https://api.typesafe.ai",
		"http://127.0.0.1:1234/prefix/api/": "http://127.0.0.1:1234/prefix/api",
	} {
		client, err := Load(func(name string) string { return map[string]string{"BASE_URL": base, "API_KEY": "key"}[name] })
		if err != nil || client.BaseURL != endpoint || client.Model != "jev-latest" {
			t.Fatalf("base %q: %+v, %v", base, client, err)
		}
	}
}

func TestProviderConfiguration(t *testing.T) {
	for _, tc := range []struct {
		provider string
		keyName  string
		endpoint string
	}{
		{"openrouter", "OPENROUTER_API_KEY", "https://openrouter.ai/api"},
		{"typesafe", "TYPESAFE_API_KEY", "https://api.typesafe.ai"},
	} {
		t.Run(tc.provider, func(t *testing.T) {
			env := map[string]string{"JEV_PROVIDER": tc.provider, tc.keyName: "provider-key", "JEV_MODEL": "jev-1.13"}
			getenv := func(name string) string { return env[name] }
			client, err := Load(getenv)
			if err != nil || client.BaseURL != tc.endpoint || client.APIKey != "provider-key" || client.Model != "jev-1.13" {
				t.Fatalf("configuración de proveedor incorrecta: %+v, %v", client, err)
			}
			env["API_KEY"], env["BASE_URL"] = "explicit-key", "https://proxy.example/base"
			client, err = Load(getenv)
			if err != nil || client.APIKey != "explicit-key" || client.BaseURL != "https://proxy.example/base" {
				t.Fatal("la configuración explícita debe prevalecer")
			}
			delete(env, "API_KEY")
			delete(env, tc.keyName)
			env["UNRELATED_API_KEY"] = "wrong-provider-key"
			if _, err := Load(getenv); err == nil {
				t.Fatal("aceptó la clave de otro proveedor")
			}
		})
	}
	if _, err := Load(func(string) string { return "unknown" }); err == nil {
		t.Fatal("aceptó un proveedor desconocido")
	}
}
