package bindmanager

import (
    "github.com/stretchr/testify/require"
    "testing"
)

func Test_SidLabel(t *testing.T) {
    tests := []struct{
        name string
        namespace string
        expected string
    }{
        {
            "default-editor",
            "jhoman",
            "AllowServiceAccountJhomanDefaultEditor",
        },
    }

    for _, subtest := range tests {
        t.Run(subtest.namespace + "/" + subtest.name, func(t *testing.T) {
            sid := SidLabel(subtest.name, subtest.namespace)
            require.Equal(t, subtest.expected, sid)
        })
    }
}
