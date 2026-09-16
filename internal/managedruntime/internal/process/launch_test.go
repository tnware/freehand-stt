package process

import "context"

func launchOwned(ctx context.Context, exe string, args []string, dir string, env []string) (*Process, error) {
	return Launch(ctx, exe, args, dir, env, Observer{})
}
