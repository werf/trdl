package docker

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	bkclient "github.com/moby/buildkit/client"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/werf/logboek"
)

const (
	buildkitContainerName        = "buildkitd"
	defaultBuildkitImage         = "moby/buildkit:buildx-stable-1"
	defaultRootlessBuildkitImage = defaultBuildkitImage + "-rootless"
	defaultBuildkitNamespace     = "default"

	// Matches the buildx kubernetes driver's own default for the same wait.
	defaultBuildkitPodTimeout = 2 * time.Minute
	buildkitPodPollInterval   = time.Second
	buildkitPodCleanupTimeout = 30 * time.Second

	// Long enough for the build to finish and the plugin to delete the pod itself,
	// so the deadline only ever fires when the plugin is no longer around to.
	buildkitPodDeadlineSlack = 5 * time.Minute
	// Fallback for a context carrying no deadline. The release task always sets
	// one, so this is what keeps the guarantee for any other caller.
	defaultBuildkitPodDeadline = time.Hour
)

// supportedKubernetesDriverOpts is this driver's own vocabulary. The names match
// the buildx kubernetes driver wherever the two overlap, but the buildx path
// passes its options through untouched while these are applied here, so one this
// driver cannot honor is rejected rather than dropped silently.
var supportedKubernetesDriverOpts = []string{
	"annotations",
	"deadline",
	"image",
	"labels",
	"limits.cpu",
	"limits.ephemeral-storage",
	"limits.memory",
	"namespace",
	"nodeselector",
	"requests.cpu",
	"requests.ephemeral-storage",
	"requests.memory",
	"rootless",
	"serviceaccount",
	"timeout",
}

type kubernetesBuilderOpts struct {
	namespace          string
	image              string
	rootless           bool
	serviceAccountName string
	nodeSelector       map[string]string
	labels             map[string]string
	annotations        map[string]string
	requests           corev1.ResourceList
	limits             corev1.ResourceList
	deadline           time.Duration
	timeout            time.Duration
}

type kubernetesBuilder struct {
	restClient   rest.Interface
	restConfig   *rest.Config
	namespace    string
	podName      string
	readyTimeout time.Duration
	logger       Logger
}

func newKubernetesBuilder(ctx context.Context, builderName, driver string, driverOpts []string, logger Logger) (*kubernetesBuilder, error) {
	if err := ValidateBuildkitdDriver(ctx, driver); err != nil {
		return nil, err
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to configure the kubernetes client: %w", err)
	}

	restClient, err := newCoreRESTClient(restConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create the kubernetes client: %w", err)
	}

	opts, err := parseKubernetesDriverOpts(trimDriverOpts(driverOpts))
	if err != nil {
		return nil, fmt.Errorf("unable to parse the buildkitd driver options: %w", err)
	}
	if opts.namespace == "" {
		opts.namespace = namespaceFromClientConfig(clientConfig)
	}

	b := &kubernetesBuilder{
		restClient:   restClient,
		restConfig:   restConfig,
		namespace:    opts.namespace,
		podName:      builderName,
		readyTimeout: opts.timeout,
		logger:       logger,
	}

	if err := b.bootstrap(ctx, opts); err != nil {
		return nil, err
	}

	return b, nil
}

// bootstrap removes whatever it created before returning an error, so a failure
// between creating the pod and it becoming ready cannot leave a builder running.
func (b *kubernetesBuilder) bootstrap(ctx context.Context, opts kubernetesBuilderOpts) error {
	opts.deadline = resolvePodDeadline(ctx, opts.deadline)

	pod := buildkitPod(b.podName, opts)

	if err := b.createPod(ctx, pod); err != nil {
		// The API server may have persisted the pod before the response was lost,
		// and the caller gets no builder back to clean up with, so the delete is
		// attempted here too. Usually nothing was created and it is a no-op, but
		// when both calls fail the operator has a privileged pod to find by hand
		// and has to be told so, the same way the readiness path tells them.
		if removeErr := b.removeAfterFailure(ctx); removeErr != nil {
			return fmt.Errorf("unable to create builder pod %s/%s: %w (a pod may have been created and was not removed: %w)", b.namespace, b.podName, err, removeErr)
		}

		return fmt.Errorf("unable to create builder pod %s/%s: %w", b.namespace, b.podName, err)
	}

	b.log(ctx, fmt.Sprintf("Waiting for builder pod %s/%s", b.namespace, b.podName))

	if err := b.waitForPod(ctx); err != nil {
		if removeErr := b.removeAfterFailure(ctx); removeErr != nil {
			return fmt.Errorf("%w (the builder pod was left behind: %w)", err, removeErr)
		}

		return err
	}

	return nil
}

// removeAfterFailure deletes the pod on a path where the caller never receives a
// builder and so cannot run the cleanup itself. A canceled build context is
// exactly when this runs, so the delete is given one of its own.
func (b *kubernetesBuilder) removeAfterFailure(ctx context.Context) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), buildkitPodCleanupTimeout)
	defer cancel()

	return b.remove(cleanupCtx)
}

// The pod's own readiness probe runs `buildctl debug workers`, so a ready pod is
// one already serving builds. Connecting earlier reaches a buildkitd that is not.
func (b *kubernetesBuilder) waitForPod(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, b.readyTimeout)
	defer cancel()

	ticker := time.NewTicker(buildkitPodPollInterval)
	defer ticker.Stop()

	lastState := ""
	for {
		pod, err := b.getPod(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("builder pod %s/%s is not ready: %w (last state: %s, last error: %v)", b.namespace, b.podName, ctx.Err(), lo.Ternary(lastState == "", "unknown", lastState), err)
			}

			// A single failed read is not a failed build: an API server rolling a
			// replica, a 429 from a fairness queue or a token rotation all produce
			// one, and the wait exists to ride them out. Keep polling until the
			// deadline, and report the last error with it.
			lastState = fmt.Sprintf("unreadable (%v)", err)

			select {
			case <-ctx.Done():
				return fmt.Errorf("builder pod %s/%s is not ready: %w (last state: %s)", b.namespace, b.podName, ctx.Err(), lastState)
			case <-ticker.C:
			}

			continue
		}

		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			return fmt.Errorf("builder pod %s/%s terminated with phase %s: %s", b.namespace, b.podName, pod.Status.Phase, pod.Status.Reason)
		}
		if isPodReady(pod) {
			return nil
		}
		lastState = podState(pod)

		select {
		case <-ctx.Done():
			return fmt.Errorf("builder pod %s/%s is not ready: %w (last state: %s)", b.namespace, b.podName, ctx.Err(), lastState)
		case <-ticker.C:
		}
	}
}

func (b *kubernetesBuilder) client(ctx context.Context) (*bkclient.Client, error) {
	client, err := bkclient.New(ctx, "", bkclient.WithContextDialer(b.dialerFor(ctx)))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to the builder pod %s/%s: %w", b.namespace, b.podName, err)
	}

	return client, nil
}

// dialerFor drops the context gRPC hands the dialer, deliberately: gRPC scopes
// it to one connection attempt and cancels it as soon as the transport is up,
// which would tear down the exec stream the connection is made of. The stream
// has to live as long as the build, so it follows the build's context.
func (b *kubernetesBuilder) dialerFor(ctx context.Context) func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) {
		return b.dial(ctx)
	}
}

func (b *kubernetesBuilder) remove(ctx context.Context) error {
	if err := b.deletePod(ctx); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("unable to delete builder pod %s/%s: %w", b.namespace, b.podName, err)
	}

	return nil
}

// The pod calls go through a plain REST client rather than the generated typed
// client, which reaches every API group and its apply configurations: linking it
// costs the plugin binary tens of megabytes to create one pod.
func newCoreRESTClient(restConfig *rest.Config) (rest.Interface, error) {
	config := rest.CopyConfig(restConfig)
	config.APIPath = "/api"
	config.GroupVersion = &corev1.SchemeGroupVersion
	config.NegotiatedSerializer = serializer.NewCodecFactory(coreScheme)
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return rest.RESTClientFor(config)
}

func (b *kubernetesBuilder) createPod(ctx context.Context, pod *corev1.Pod) error {
	return b.restClient.Post().
		Namespace(b.namespace).
		Resource("pods").
		Body(pod).
		Do(ctx).
		Error()
}

func (b *kubernetesBuilder) getPod(ctx context.Context) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	if err := b.restClient.Get().
		Namespace(b.namespace).
		Resource("pods").
		Name(b.podName).
		Do(ctx).
		Into(pod); err != nil {
		return nil, err
	}

	return pod, nil
}

func (b *kubernetesBuilder) deletePod(ctx context.Context) error {
	return b.restClient.Delete().
		Namespace(b.namespace).
		Resource("pods").
		Name(b.podName).
		Do(ctx).
		Error()
}

// coreScheme carries core/v1 alone. client-go's own kubernetes/scheme registers
// every API group, which links the whole generated surface into the plugin.
var coreScheme = newCoreScheme()

func newCoreScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	lo.Must0(corev1.AddToScheme(scheme))

	return scheme
}

func (b *kubernetesBuilder) log(ctx context.Context, msg string) {
	logboek.Context(ctx).Default().LogLn(msg)
	b.logger.Info(msg)
}

func isPodReady(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}

	condition, found := lo.Find(pod.Status.Conditions, func(c corev1.PodCondition) bool {
		return c.Type == corev1.PodReady
	})

	return found && condition.Status == corev1.ConditionTrue
}

// podState is only ever shown to an operator whose build is stuck waiting, so it
// reports whatever the pod says about itself rather than a fixed set of reasons.
func podState(pod *corev1.Pod) string {
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Waiting != nil && status.State.Waiting.Reason != "" {
			return fmt.Sprintf("%s (%s)", pod.Status.Phase, status.State.Waiting.Reason)
		}
	}

	return string(pod.Status.Phase)
}

func namespaceFromClientConfig(clientConfig clientcmd.ClientConfig) string {
	namespace, _, err := clientConfig.Namespace()
	if err != nil || strings.TrimSpace(namespace) == "" {
		return defaultBuildkitNamespace
	}

	return namespace
}

func parseKubernetesDriverOpts(driverOpts []string) (kubernetesBuilderOpts, error) {
	opts := kubernetesBuilderOpts{
		nodeSelector: map[string]string{},
		labels:       map[string]string{},
		annotations:  map[string]string{},
		requests:     corev1.ResourceList{},
		limits:       corev1.ResourceList{},
	}

	imageSet, timeoutSet, deadlineSet := false, false, false
	for _, driverOpt := range driverOpts {
		name, value, found := strings.Cut(driverOpt, "=")
		if !found {
			return opts, fmt.Errorf("driver option %q is not a name=value pair", driverOpt)
		}
		name = strings.TrimSpace(name)

		var err error
		switch name {
		case "namespace":
			opts.namespace = strings.TrimSpace(value)
		case "image":
			// A blank value means "not set" here as everywhere else, so it must not
			// suppress the rootless default and store an image the API server will
			// refuse.
			opts.image = strings.TrimSpace(value)
			imageSet = opts.image != ""
		case "serviceaccount":
			opts.serviceAccountName = strings.TrimSpace(value)
		case "rootless":
			opts.rootless, err = strconv.ParseBool(strings.TrimSpace(value))
		case "deadline":
			deadlineSet = true
			opts.deadline, err = time.ParseDuration(strings.TrimSpace(value))
		case "timeout":
			timeoutSet = true
			opts.timeout, err = time.ParseDuration(strings.TrimSpace(value))
		case "nodeselector":
			err = mergeKeyValues(opts.nodeSelector, value)
		case "labels":
			err = mergeKeyValues(opts.labels, value)
		case "annotations":
			err = mergeKeyValues(opts.annotations, value)
		case "requests.cpu", "requests.memory", "requests.ephemeral-storage":
			err = setResourceQuantity(opts.requests, strings.TrimPrefix(name, "requests."), value)
		case "limits.cpu", "limits.memory", "limits.ephemeral-storage":
			err = setResourceQuantity(opts.limits, strings.TrimPrefix(name, "limits."), value)
		default:
			return opts, fmt.Errorf("unsupported option %q for the %s buildkitd driver (supported: %s)", name, buildkitdDriverKubernetes, strings.Join(supportedKubernetesDriverOpts, ", "))
		}
		if err != nil {
			return opts, fmt.Errorf("driver option %q: %w", name, err)
		}
	}

	if !imageSet {
		opts.image = lo.Ternary(opts.rootless, defaultRootlessBuildkitImage, defaultBuildkitImage)
	}
	if !timeoutSet {
		opts.timeout = defaultBuildkitPodTimeout
	}
	// activeDeadlineSeconds is whole seconds and must be at least one, so a
	// sub-second deadline would truncate to zero and a fractional one would
	// silently lose its remainder.
	if deadlineSet && (opts.deadline < time.Second || opts.deadline%time.Second != 0) {
		return opts, fmt.Errorf("driver option %q: must be a whole number of seconds, at least 1s", "deadline")
	}
	if opts.timeout <= 0 {
		return opts, fmt.Errorf("driver option %q: must be positive", "timeout")
	}
	for name, request := range opts.requests {
		limit, ok := opts.limits[name]
		if ok && request.Cmp(limit) > 0 {
			return opts, fmt.Errorf("driver option %q: %s exceeds the %s limit of %s", "requests."+string(name), request.String(), name, limit.String())
		}
	}

	return opts, nil
}

func setResourceQuantity(list corev1.ResourceList, name, value string) error {
	quantity, err := resource.ParseQuantity(strings.TrimSpace(value))
	if err != nil {
		return err
	}
	// ParseQuantity accepts a signed quantity, and a negative one is only rejected
	// by the API server when the build finally tries to create the pod.
	if quantity.Sign() < 0 {
		return fmt.Errorf("must not be negative")
	}
	list[corev1.ResourceName(name)] = quantity

	return nil
}

// mergeKeyValues adds into the map rather than replacing it: the options are one
// name=value pair per element, so an operator naturally writes the same option
// twice, and replacing would drop the earlier pairs without an error anywhere.
func mergeKeyValues(into map[string]string, value string) error {
	parsed, err := splitKeyValues(value)
	if err != nil {
		return err
	}
	for k, v := range parsed {
		into[k] = v
	}

	return nil
}

func splitKeyValues(value string) (map[string]string, error) {
	result := map[string]string{}
	for _, pair := range strings.Split(value, ",") {
		if strings.TrimSpace(pair) == "" {
			continue
		}

		name, v, found := strings.Cut(pair, "=")
		if !found {
			return nil, fmt.Errorf("%q is not a name=value pair", pair)
		}
		result[strings.TrimSpace(name)] = strings.TrimSpace(v)
	}

	return result, nil
}

// resolvePodDeadline bounds the pod's lifetime by the build's own, because
// nothing outside the plugin process deletes the builder: a crash between
// creating the pod and removing it would otherwise leave it running forever.
// The floor keeps an already-expired context from producing a deadline the API
// server rejects, and the whole-second rounding matches what the option itself
// is validated against.
func resolvePodDeadline(ctx context.Context, configured time.Duration) time.Duration {
	if configured > 0 {
		return configured
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		return defaultBuildkitPodDeadline
	}

	return max(time.Until(deadline).Round(time.Second), time.Minute) + buildkitPodDeadlineSlack
}

func buildkitPod(name string, opts kubernetesBuilderOpts) *corev1.Pod {
	labels := map[string]string{"app": name}
	for k, v := range opts.labels {
		labels[k] = v
	}

	annotations := map[string]string{}
	for k, v := range opts.annotations {
		annotations[k] = v
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   opts.namespace,
			Name:        name,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			RestartPolicy:                corev1.RestartPolicyNever,
			ServiceAccountName:           opts.serviceAccountName,
			AutomountServiceAccountToken: lo.ToPtr(false),
			NodeSelector:                 opts.nodeSelector,
			Containers: []corev1.Container{
				{
					Name:  buildkitContainerName,
					Image: opts.image,
					SecurityContext: &corev1.SecurityContext{
						Privileged: lo.ToPtr(true),
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							Exec: &corev1.ExecAction{
								Command: []string{"buildctl", "debug", "workers"},
							},
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: opts.requests,
						Limits:   opts.limits,
					},
				},
			},
		},
	}

	if opts.deadline > 0 {
		pod.Spec.ActiveDeadlineSeconds = lo.ToPtr(int64(opts.deadline.Seconds()))
	}
	if opts.rootless {
		toRootless(pod)
	}

	return pod
}

// Rootless BuildKit needs the whole set: dropping the seccomp profile or the
// AppArmor annotation gives a pod that starts and then fails every build.
func toRootless(pod *corev1.Pod) {
	container := &pod.Spec.Containers[0]

	container.Args = append(container.Args, "--oci-worker-no-process-sandbox")
	container.SecurityContext = &corev1.SecurityContext{
		SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeUnconfined},
		// The field is the supported API; the annotation below is deprecated since
		// Kubernetes 1.30 and carried only so that clusters older than that still
		// see the profile.
		AppArmorProfile: &corev1.AppArmorProfile{Type: corev1.AppArmorProfileTypeUnconfined},
	}
	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name: buildkitContainerName,
		// The image declares this path as a VOLUME, which rootless cannot use on
		// hosts mounting it nosuid,nodev; an emptyDir replaces it.
		MountPath: "/home/user/.local/share/buildkit",
	})

	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name:         buildkitContainerName,
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	})
	pod.Annotations["container.apparmor.security.beta.kubernetes.io/"+buildkitContainerName] = "unconfined"
}
