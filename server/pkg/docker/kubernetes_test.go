package docker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"

	"github.com/werf/logboek"
)

// fakeAPIServer answers the three pod calls the builder makes. Tests drive a
// builder against it instead of a cluster, so a mutation that removes a guard
// reaches this server and fails the assertion rather than provisioning anything.
type fakeAPIServer struct {
	*httptest.Server

	mu         sync.Mutex
	calls      []string
	created    *corev1.Pod
	deleted    bool
	readyPod   bool
	phase      corev1.PodPhase
	failCreate bool
	failDelete bool
}

func newFakeAPIServer(t *testing.T) *fakeAPIServer {
	f := &fakeAPIServer{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		f.calls = append(f.calls, r.Method+" "+r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")

		// The exec subresource is a POST with no JSON body and a SPDY upgrade this
		// server does not speak; it is recorded and refused, which is enough to
		// show the request was issued at all.
		if strings.HasSuffix(r.URL.Path, "/exec") {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		switch r.Method {
		case http.MethodPost:
			pod := &corev1.Pod{}
			require.NoError(t, json.NewDecoder(r.Body).Decode(pod))
			// The pod is recorded before the status is written, so failCreate
			// reproduces the case where the API server persisted it and the
			// client saw only an error.
			f.created = pod
			if f.failCreate {
				w.WriteHeader(http.StatusInternalServerError)
				require.NoError(t, json.NewEncoder(w).Encode(&metav1.Status{Status: metav1.StatusFailure, Code: http.StatusInternalServerError}))

				return
			}
			w.WriteHeader(http.StatusCreated)
			require.NoError(t, json.NewEncoder(w).Encode(pod))
		case http.MethodGet:
			if f.created == nil || f.deleted {
				writeNotFound(t, w)

				return
			}
			require.NoError(t, json.NewEncoder(w).Encode(podWithStatus(f.created, f.readyPod, f.phase)))
		case http.MethodDelete:
			if f.failDelete {
				w.WriteHeader(http.StatusInternalServerError)
				require.NoError(t, json.NewEncoder(w).Encode(&metav1.Status{Status: metav1.StatusFailure, Code: http.StatusInternalServerError}))

				return
			}
			if f.created == nil || f.deleted {
				writeNotFound(t, w)

				return
			}
			f.deleted = true
			require.NoError(t, json.NewEncoder(w).Encode(&metav1.Status{Status: metav1.StatusSuccess}))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(f.Close)

	return f
}

func writeNotFound(t *testing.T, w http.ResponseWriter) {
	t.Helper()

	w.WriteHeader(http.StatusNotFound)
	require.NoError(t, json.NewEncoder(w).Encode(&metav1.Status{
		Status: metav1.StatusFailure,
		Code:   http.StatusNotFound,
		Reason: metav1.StatusReasonNotFound,
	}))
}

func podWithStatus(pod *corev1.Pod, ready bool, phase corev1.PodPhase) *corev1.Pod {
	result := pod.DeepCopy()
	result.Status.Phase = corev1.PodPending
	if phase != "" {
		result.Status.Phase = phase

		return result
	}
	if ready {
		result.Status.Phase = corev1.PodRunning
		result.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}
	}

	return result
}

func (f *fakeAPIServer) methods() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	methods := make([]string, 0, len(f.calls))
	for _, call := range f.calls {
		methods = append(methods, strings.Fields(call)[0])
	}

	return methods
}

func (f *fakeAPIServer) recordedCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.calls...)
}

func (f *fakeAPIServer) createdPod() *corev1.Pod {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.created.DeepCopy()
}

func (f *fakeAPIServer) podDeleted() bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.deleted
}

func newTestBuilder(t *testing.T, f *fakeAPIServer, readyTimeout time.Duration) *kubernetesBuilder {
	t.Helper()

	// This path provisions the builder itself and must never reach for a binary.
	// An empty PATH keeps a mutation that reintroduces one from finding a real
	// docker or kubectl on the machine running the tests.
	t.Setenv("PATH", "")

	config := &rest.Config{Host: f.URL}
	config.APIPath = "/api"
	config.GroupVersion = &corev1.SchemeGroupVersion
	config.NegotiatedSerializer = serializer.NewCodecFactory(coreScheme)

	restClient, err := rest.RESTClientFor(config)
	require.NoError(t, err)

	return &kubernetesBuilder{
		restClient:   restClient,
		restConfig:   config,
		namespace:    "trdl-build",
		podName:      "trdl-builder-42",
		readyTimeout: readyTimeout,
		logger:       smokeLogger{t},
	}
}

func testContext() context.Context {
	return logboek.NewContext(context.Background(), logboek.DefaultLogger())
}

func contextWithDeadlineIn(t *testing.T, parent context.Context, d time.Duration) context.Context {
	t.Helper()

	ctx, cancel := context.WithDeadline(parent, time.Now().Add(d))
	t.Cleanup(cancel)

	return ctx
}

func testBuilderOpts() kubernetesBuilderOpts {
	opts, err := parseKubernetesDriverOpts([]string{"namespace=trdl-build"})
	if err != nil {
		panic(err)
	}

	return opts
}

func TestKubernetesBuilderBootstrapReady(t *testing.T) {
	f := newFakeAPIServer(t)
	f.readyPod = true

	b := newTestBuilder(t, f, time.Minute)

	require.NoError(t, b.bootstrap(testContext(), testBuilderOpts()))
	assert.False(t, f.podDeleted(), "a builder that came up must not be removed by bootstrap")
	assert.Equal(t, []string{"POST", "GET"}, f.methods())
}

// The pod must not outlive a bootstrap that failed: the caller gets no builder
// back, so nothing else is in a position to remove it.
func TestKubernetesBuilderBootstrapRemovesPodThatNeverBecomesReady(t *testing.T) {
	f := newFakeAPIServer(t)
	f.readyPod = false

	b := newTestBuilder(t, f, 100*time.Millisecond)

	err := b.bootstrap(testContext(), testBuilderOpts())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not ready")
	assert.True(t, f.podDeleted(), "the builder pod must be removed when it never becomes ready")
}

// Canceling the release is the case the cleanup exists for, and the case where
// reusing the build context would silently skip the delete.
func TestKubernetesBuilderBootstrapRemovesPodOnCancelledContext(t *testing.T) {
	f := newFakeAPIServer(t)
	f.readyPod = false

	b := newTestBuilder(t, f, time.Minute)

	ctx, cancel := context.WithCancel(testContext())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := b.bootstrap(ctx, testBuilderOpts())

	require.Error(t, err)
	assert.True(t, f.podDeleted(), "the builder pod must be removed after the build context is canceled")
}

// A Running pod is not a serving one: buildkitd reports itself through the
// readiness probe, and connecting before it passes reaches a daemon that is not
// yet accepting builds.
func TestKubernetesBuilderBootstrapWaitsForReadinessNotJustRunning(t *testing.T) {
	f := newFakeAPIServer(t)
	f.phase = corev1.PodRunning

	b := newTestBuilder(t, f, 100*time.Millisecond)

	err := b.bootstrap(testContext(), testBuilderOpts())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not ready")
	assert.True(t, f.podDeleted(), "the builder pod must be removed when readiness never arrives")
}

// A pod the API server persisted while the client saw a failure is invisible to
// the caller — no builder is returned, so nothing else can clean it up.
func TestKubernetesBuilderBootstrapRemovesPodAfterFailedCreate(t *testing.T) {
	f := newFakeAPIServer(t)
	f.failCreate = true

	b := newTestBuilder(t, f, time.Minute)

	err := b.bootstrap(testContext(), testBuilderOpts())

	require.Error(t, err)
	assert.True(t, f.podDeleted(), "a pod that may have been created must be deleted when create reports failure")
	// The forwarding guard in CI greps this message for the namespace the driver
	// options carried, so the namespace has to be in it.
	assert.Contains(t, err.Error(), "builder pod trdl-build/", "the error must name the namespace the builder was configured with")
}

// When the create fails and the cleanup fails too, the operator has a privileged
// pod nobody can see and has to be told: the create error alone does not say a
// pod may exist.
func TestKubernetesBuilderBootstrapReportsAPodItCouldNotRemove(t *testing.T) {
	f := newFakeAPIServer(t)
	f.failCreate = true
	f.failDelete = true

	b := newTestBuilder(t, f, time.Minute)

	err := b.bootstrap(testContext(), testBuilderOpts())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to create builder pod")
	assert.Contains(t, err.Error(), "was not removed", "a pod that may exist and could not be deleted has to be named")
}

// gRPC cancels the context it hands a dialer as soon as the transport is up. If
// the exec stream were scoped to that context it would be torn down immediately,
// which is what made the first cluster run fail with "context canceled". The
// stream must follow the build's context instead: with a dead dialer context the
// request still has to reach the API server.
func TestKubernetesBuilderClientIgnoresDialerContext(t *testing.T) {
	f := newFakeAPIServer(t)
	f.readyPod = true

	b := newTestBuilder(t, f, time.Minute)

	dead, cancel := context.WithCancel(context.Background())
	cancel()

	conn, err := b.dialerFor(testContext())(dead, "")
	if err == nil {
		t.Cleanup(func() { _ = conn.Close() })
	}

	assert.Eventually(t, func() bool {
		for _, call := range f.recordedCalls() {
			if strings.Contains(call, "/exec") {
				return true
			}
		}

		return false
	}, 5*time.Second, 20*time.Millisecond, "the exec request must be issued even when the dialer context is already canceled")
}

func TestKubernetesBuilderRemoveToleratesMissingPod(t *testing.T) {
	f := newFakeAPIServer(t)
	b := newTestBuilder(t, f, time.Minute)

	assert.NoError(t, b.remove(testContext()), "removing a pod that is already gone is not a failure")
}

func TestKubernetesBuilderBootstrapRemovesTerminatedPod(t *testing.T) {
	f := newFakeAPIServer(t)
	f.phase = corev1.PodFailed

	b := newTestBuilder(t, f, time.Minute)

	err := b.bootstrap(testContext(), testBuilderOpts())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "terminated")
	assert.True(t, f.podDeleted(), "a builder pod that died must be removed, not left for an operator")
}

// The exec command is the whole transport: changing it to anything else breaks
// every build, and nothing in the required suite would notice if it were not
// asserted here.
func TestKubernetesBuilderDialsBuildctlDialStdio(t *testing.T) {
	f := newFakeAPIServer(t)
	f.readyPod = true

	b := newTestBuilder(t, f, time.Minute)

	conn, err := b.dialerFor(testContext())(testContext(), "")
	if err == nil {
		t.Cleanup(func() { _ = conn.Close() })
	}

	assert.Eventually(t, func() bool {
		for _, call := range f.recordedCalls() {
			if strings.Contains(call, "command=buildctl") && strings.Contains(call, "command=dial-stdio") {
				return true
			}
		}

		return false
	}, 5*time.Second, 20*time.Millisecond, "the exec request must run buildctl dial-stdio in the builder container")
}

// Remove on the happy path — a builder that came up and is torn down after a
// successful build — is what the required suite never reaches, so it is asserted
// here rather than left to the opt-in cluster job.
func TestBuilderRemoveDeletesTheProvisionedPod(t *testing.T) {
	f := newFakeAPIServer(t)
	f.readyPod = true

	kb := newTestBuilder(t, f, time.Minute)
	require.NoError(t, kb.bootstrap(testContext(), testBuilderOpts()))
	require.False(t, f.podDeleted())

	builder := &Builder{builderName: kb.podName, kubernetesBuilder: kb, logger: smokeLogger{t}}

	require.NoError(t, builder.Remove(testContext()))
	assert.True(t, f.podDeleted(), "removing the builder must delete the pod it provisioned")
}

// A pod outliving the plugin process is the case this covers: nothing deletes
// the builder if the plugin is killed mid-build, so every pod must carry a
// deadline even when the operator configured none.
func TestResolvePodDeadline(t *testing.T) {
	for name, tc := range map[string]struct {
		ctx        context.Context
		configured time.Duration
		expected   time.Duration
	}{
		"a configured deadline wins over the context": {
			ctx:        contextWithDeadlineIn(t, context.Background(), 10*time.Minute),
			configured: 90 * time.Minute,
			expected:   90 * time.Minute,
		},
		"the remaining task time plus the slack": {
			ctx:      contextWithDeadlineIn(t, context.Background(), 30*time.Minute),
			expected: 30*time.Minute + buildkitPodDeadlineSlack,
		},
		"an expired context cannot produce a negative deadline": {
			ctx:      contextWithDeadlineIn(t, context.Background(), -time.Hour),
			expected: buildkitPodDeadlineSlack,
		},
		"a context without a deadline falls back": {
			ctx:      context.Background(),
			expected: defaultBuildkitPodDeadline,
		},
	} {
		t.Run(name, func(t *testing.T) {
			resolved := resolvePodDeadline(tc.ctx, tc.configured)

			// A deadline derived from the clock loses up to a second to rounding
			// between the context being built and this call, so the tolerance is the
			// rounding, not the property: the whole-second assertion below is what
			// catches a value that skipped rounding altogether.
			assert.InDelta(t, tc.expected, resolved, float64(2*time.Second))
			assert.Zero(t, resolved%time.Second, "activeDeadlineSeconds is whole seconds")
			assert.GreaterOrEqual(t, resolved, time.Second, "activeDeadlineSeconds must be at least 1s")
		})
	}
}

// The manifest is read back from the fake API server rather than from
// buildkitPod, because buildkitPod has no context and only serializes the
// deadline it is handed: dropping the resolve call would leave it nil here.
func TestKubernetesBuilderBootstrapAlwaysSetsAPodDeadline(t *testing.T) {
	f := newFakeAPIServer(t)
	f.readyPod = true

	b := newTestBuilder(t, f, time.Minute)
	ctx := contextWithDeadlineIn(t, testContext(), 30*time.Minute)

	require.NoError(t, b.bootstrap(ctx, testBuilderOpts()))

	deadline := f.createdPod().Spec.ActiveDeadlineSeconds
	require.NotNil(t, deadline, "bootstrap must give every builder pod a deadline")
	assert.InDelta(t, int64((30*time.Minute + buildkitPodDeadlineSlack).Seconds()), *deadline, 5)
}

func TestBuildkitPodDefaults(t *testing.T) {
	pod := buildkitPod("trdl-builder-42", testBuilderOpts())

	require.Len(t, pod.Spec.Containers, 1)
	container := pod.Spec.Containers[0]

	assert.Equal(t, corev1.RestartPolicyNever, pod.Spec.RestartPolicy)
	assert.Equal(t, defaultBuildkitImage, container.Image)
	assert.True(t, *container.SecurityContext.Privileged)
	assert.Equal(t, []string{"buildctl", "debug", "workers"}, container.ReadinessProbe.Exec.Command)
	assert.Nil(t, pod.Spec.ActiveDeadlineSeconds, "buildkitPod only serializes a deadline it is given; bootstrap is what defaults it")
	assert.Equal(t, lo.ToPtr(false), pod.Spec.AutomountServiceAccountToken, "a privileged pod running project instructions must not get the namespace's default token")
	assert.Equal(t, map[string]string{"app": "trdl-builder-42"}, pod.Labels)
}

func TestBuildkitPodRootless(t *testing.T) {
	opts, err := parseKubernetesDriverOpts([]string{"rootless=true"})
	require.NoError(t, err)

	pod := buildkitPod("trdl-builder-42", opts)
	container := pod.Spec.Containers[0]

	assert.Equal(t, defaultRootlessBuildkitImage, container.Image)
	assert.Contains(t, container.Args, "--oci-worker-no-process-sandbox")
	assert.Nil(t, container.SecurityContext.Privileged, "rootless must not ask for a privileged container")
	assert.Equal(t, corev1.SeccompProfileTypeUnconfined, container.SecurityContext.SeccompProfile.Type)
	// The annotation alone is not enough on a cluster that no longer converts it
	// into the field, so both have to be present.
	assert.Equal(t, corev1.AppArmorProfileTypeUnconfined, container.SecurityContext.AppArmorProfile.Type)
	assert.Equal(t, "unconfined", pod.Annotations["container.apparmor.security.beta.kubernetes.io/buildkitd"])
	assert.Len(t, pod.Spec.Volumes, 1)
	assert.Len(t, container.VolumeMounts, 1)
}

func TestBuildkitPodRootlessKeepsConfiguredImage(t *testing.T) {
	opts, err := parseKubernetesDriverOpts([]string{"rootless=true", "image=registry.example.com/buildkit:v0.31.2-rootless"})
	require.NoError(t, err)

	assert.Equal(t, "registry.example.com/buildkit:v0.31.2-rootless", buildkitPod("trdl-builder-42", opts).Spec.Containers[0].Image)
}

func TestBuildkitPodAppliesResourcesAndScheduling(t *testing.T) {
	opts, err := parseKubernetesDriverOpts([]string{
		"requests.cpu=500m",
		"limits.memory=4Gi",
		"nodeselector=disktype=ssd,zone=a",
		"serviceaccount=trdl-buildkit",
		"deadline=90m",
		"labels=team=delivery",
		"annotations=example.com/owner=trdl",
	})
	require.NoError(t, err)

	pod := buildkitPod("trdl-builder-42", opts)
	container := pod.Spec.Containers[0]

	assert.Equal(t, "500m", container.Resources.Requests.Cpu().String())
	assert.Equal(t, "4Gi", container.Resources.Limits.Memory().String())
	assert.Equal(t, map[string]string{"disktype": "ssd", "zone": "a"}, pod.Spec.NodeSelector)
	assert.Equal(t, "trdl-buildkit", pod.Spec.ServiceAccountName)
	assert.Equal(t, lo.ToPtr(false), pod.Spec.AutomountServiceAccountToken, "a configured ServiceAccount must not hand its token to a privileged build container")
	assert.Equal(t, int64(5400), *pod.Spec.ActiveDeadlineSeconds)
	assert.Equal(t, "delivery", pod.Labels["team"])
	assert.Equal(t, "trdl", pod.Annotations["example.com/owner"])
}

// One name=value pair per element is what the field's own description asks for,
// so the same option written twice is a natural shape — and replacing instead of
// merging would drop the earlier pairs with no error anywhere.
func TestParseKubernetesDriverOptsMergesRepeatedMapOptions(t *testing.T) {
	opts, err := parseKubernetesDriverOpts([]string{
		"nodeselector=kubernetes.io/arch=amd64",
		"nodeselector=workload=build",
		"labels=team=delivery",
		"labels=cost-center=42",
	})

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"kubernetes.io/arch": "amd64", "workload": "build"}, opts.nodeSelector)
	assert.Equal(t, map[string]string{"team": "delivery", "cost-center": "42"}, opts.labels)
}

// A blank image must mean "not set", or it silently suppresses the rootless
// default and stores a pod spec the API server refuses at release time.
func TestParseKubernetesDriverOptsBlankImageKeepsDefault(t *testing.T) {
	opts, err := parseKubernetesDriverOpts([]string{"rootless=true", "image=   "})

	require.NoError(t, err)
	assert.Equal(t, defaultRootlessBuildkitImage, opts.image)
}

func TestParseKubernetesDriverOptsRejections(t *testing.T) {
	for name, driverOpts := range map[string][]string{
		"unsupported option":    {"tolerations=key=node,operator=Exists"},
		"not a pair":            {"namespace"},
		"bad boolean":           {"rootless=yes-please"},
		"bad duration":          {"deadline=90"},
		"bad quantity":          {"limits.memory=4 gigabytes"},
		"negative deadline":     {"deadline=-1m"},
		"sub-second deadline":   {"deadline=500ms"},
		"truncating deadline":   {"deadline=1500ms"},
		"padded bad duration":   {"deadline= 90"},
		"negative cpu request":  {"requests.cpu=-1"},
		"negative memory limit": {"limits.memory=-500Mi"},
		"non-positive timeout":  {"timeout=0s"},
	} {
		rejected := driverOpts
		t.Run(name, func(t *testing.T) {
			_, err := parseKubernetesDriverOpts(rejected)
			assert.Error(t, err)
		})
	}
}

// The sibling arms of the same switch trim their values, so a padded pair is a
// shape the parser already accepts elsewhere — rejecting it here only for these
// three options would be an error the operator cannot see in their own config.
func TestParseKubernetesDriverOptsTrimsPaddedValues(t *testing.T) {
	opts, err := parseKubernetesDriverOpts([]string{"rootless= true", "deadline= 90m", "timeout= 10s"})

	require.NoError(t, err)
	assert.True(t, opts.rootless)
	assert.Equal(t, 90*time.Minute, opts.deadline)
	assert.Equal(t, 10*time.Second, opts.timeout)
}

func TestParseKubernetesDriverOptsTimeoutDefaults(t *testing.T) {
	opts, err := parseKubernetesDriverOpts(nil)

	require.NoError(t, err)
	assert.Equal(t, defaultBuildkitPodTimeout, opts.timeout)

	opts, err = parseKubernetesDriverOpts([]string{"timeout=10s"})

	require.NoError(t, err)
	assert.Equal(t, 10*time.Second, opts.timeout)
}

func TestValidateBuildkitdDriverOpts(t *testing.T) {
	assert.NoError(t, ValidateBuildkitdDriverOpts(context.Background(), "", nil))
	assert.NoError(t, ValidateBuildkitdDriverOpts(context.Background(), "", []string{"   "}))
	assert.Error(t, ValidateBuildkitdDriverOpts(context.Background(), "", []string{"namespace=trdl-build"}),
		"options without a driver would never be applied")
	assert.NoError(t, ValidateBuildkitdDriverOpts(context.Background(), "kubernetes", []string{"namespace=trdl-build"}))
	assert.Error(t, ValidateBuildkitdDriverOpts(context.Background(), "kubernetes", []string{"replicas=3"}))
}

func TestValidateBuildkitdDriver(t *testing.T) {
	assert.NoError(t, ValidateBuildkitdDriver(context.Background(), ""))
	assert.NoError(t, ValidateBuildkitdDriver(context.Background(), "kubernetes"))
	assert.Error(t, ValidateBuildkitdDriver(context.Background(), "docker-container"))
}
