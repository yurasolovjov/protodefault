package protodefault

import (
	"testing"

	"github.com/stretchr/testify/require"

	testv1 "github.com/yurasolovjov/protodefault/proto/test/v1"
)

func TestApply_Scalars(t *testing.T) {
	msg := &testv1.TestMessage{}
	err := Apply(msg)
	require.NoError(t, err)

	require.Equal(t, "default string", msg.StrField)
	require.Equal(t, int32(42), msg.Int32Field)
	require.Equal(t, uint64(100500), msg.Uint64Field)
	require.Equal(t, 3.14, msg.DoubleField)
	require.Equal(t, true, msg.BoolField)
	require.Equal(t, []byte("hello"), msg.BytesField)

	require.Equal(t, "optional default", *msg.OptStrField)
	require.Equal(t, int32(-1), *msg.OptInt32Field)
}

func TestApply_Enum(t *testing.T) {
	msg := &testv1.TestMessage{}
	err := Apply(msg)
	require.NoError(t, err)

	require.Equal(t, testv1.LogLevel_LOG_LEVEL_INFO, msg.LevelByName)
	require.Equal(t, testv1.LogLevel_LOG_LEVEL_WARN, msg.LevelByNumber)
}

func TestApply_Nested(t *testing.T) {
	t.Run("auto-initialize", func(t *testing.T) {
		msg := &testv1.TestMessage{}
		err := Apply(msg)
		require.NoError(t, err)
		require.NotNil(t, msg.Feature)
		require.Equal(t, "unknown", msg.Feature.Name)
		require.Equal(t, true, msg.Feature.Enabled)
	})

	t.Run("do-not-overwrite", func(t *testing.T) {
		msg := &testv1.TestMessage{
			Feature: &testv1.FeatureFlag{
				Name: "custom",
			},
		}
		err := Apply(msg)
		require.NoError(t, err)
		require.Equal(t, "custom", msg.Feature.Name)
		require.Equal(t, true, msg.Feature.Enabled) // this should still be applied because it was zero
	})
}

func TestApply_Repeated(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		msg := &testv1.TestMessage{}
		err := Apply(msg)
		require.NoError(t, err)
		require.Equal(t, []string{"tag1", "tag2"}, msg.Tags)
		require.Len(t, msg.Features, 1)
		require.Equal(t, "feat1", msg.Features[0].Name)
	})

	t.Run("not-empty", func(t *testing.T) {
		msg := &testv1.TestMessage{
			Tags: []string{"custom"},
		}
		err := Apply(msg)
		require.NoError(t, err)
		require.Equal(t, []string{"custom"}, msg.Tags)
	})
}

func TestApply_Map(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		msg := &testv1.TestMessage{}
		err := Apply(msg)
		require.NoError(t, err)
		require.Equal(t, int32(10), msg.StringToInt["key1"])
		require.NotNil(t, msg.StringToMsg["x"])
		// FeatureFlag has enabled=true default. Even if JSON says false,
		// because false is zero-value in proto3, it's overwritten by default.
		require.Equal(t, true, msg.StringToMsg["x"].Enabled)
		require.Equal(t, "unknown", msg.StringToMsg["x"].Name)
	})

	t.Run("not-empty", func(t *testing.T) {
		msg := &testv1.TestMessage{
			StringToInt: map[string]int32{"custom": 1},
		}
		err := Apply(msg)
		require.NoError(t, err)
		require.Len(t, msg.StringToInt, 1)
	})
}

func TestApply_Oneof(t *testing.T) {
	t.Run("default-choice", func(t *testing.T) {
		msg := &testv1.TestMessage{}
		err := Apply(msg)
		require.NoError(t, err)
		require.Equal(t, "chosen", msg.GetChoiceStr())
	})

	t.Run("user-choice", func(t *testing.T) {
		msg := &testv1.TestMessage{
			Choice: &testv1.TestMessage_ChoiceInt{ChoiceInt: 123},
		}
		err := Apply(msg)
		require.NoError(t, err)
		require.Equal(t, int32(123), msg.GetChoiceInt())
		require.Equal(t, "", msg.GetChoiceStr())
	})

	t.Run("conflict", func(t *testing.T) {
		msg := &testv1.OneofConflict{}
		err := Apply(msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "multiple fields with default_value")
	})
}

func TestApply_WKT(t *testing.T) {
	msg := &testv1.TestMessage{}
	err := Apply(msg)
	require.NoError(t, err)

	require.NotNil(t, msg.Duration, "Duration should be initialized")
	require.Equal(t, int64(15), msg.Duration.Seconds)
	require.Equal(t, int64(1767225600), msg.Timestamp.Seconds) // 2026-01-01
	require.Equal(t, "val", msg.StructField.Fields["key"].GetStringValue())
	require.Equal(t, 2, len(msg.ListField.Values))
	require.Equal(t, "any value", msg.ValueField.GetStringValue())
	require.Equal(t, []string{"a.b", "c.d"}, msg.MaskField.Paths)
	require.Equal(t, true, msg.BoolWrapper.Value)
}

func TestApply_Recursive(t *testing.T) {
	msg := &testv1.RecursiveA{
		B: &testv1.RecursiveB{
			A: &testv1.RecursiveA{},
		},
	}
	err := Apply(msg)
	require.NoError(t, err) // Should not stack overflow
}

func TestApply_ZeroValueOverwriting(t *testing.T) {
	// proto3 scalar zero-value IS overwritten by default because we can't distinguish
	msg := &testv1.TestMessage{
		Int32Field: 0,
	}
	err := Apply(msg)
	require.NoError(t, err)
	require.Equal(t, int32(42), msg.Int32Field) // Limitation of proto3
}

func TestApply_NilInput(t *testing.T) {
	err := Apply(nil)
	require.Error(t, err)
}
