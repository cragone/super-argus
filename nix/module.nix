# NixOS module for the super-argus server + agent, exposed as
# `nixosModules.default` from this flake. Built by an activation script
# rather than a buildGoModule derivation: no vendorHash to keep in sync, and
# the server also needs a Node/Expo build step (app/) that doesn't fit
# buildGoModule's sandboxed model anyway. `argusSrc` is this flake's own
# `self`, copied into a writable build dir each activation so a `nixos-rebuild
# switch` against a newer flake lock always rebuilds against that commit.
{ argusSrc }:
{ config, lib, pkgs, ... }:

let
	cfg = config.services.argus;
	buildDir = "/var/cache/super-argus-build";
	srcDir = "${buildDir}/src";
	stateDir = "/var/lib/super-argus";
	binDir = "${stateDir}/bin";
	goCache = "/var/cache/super-argus-go";
	addr = "${cfg.host}:${toString cfg.port}";
in {
	options.services.argus = {
		enable = lib.mkEnableOption "the super-argus server";

		host = lib.mkOption {
			type = lib.types.str;
			default = "127.0.0.1";
			description = "Bind host for the Argus server.";
		};

		port = lib.mkOption {
			type = lib.types.port;
			default = 9100;
			description = "Bind port for the Argus server.";
		};

		dataDir = lib.mkOption {
			type = lib.types.str;
			default = "${stateDir}/data";
			description = "Where the Argus server's SQLite database and log chunks live.";
		};

		environmentFile = lib.mkOption {
			type = lib.types.str;
			default = "/etc/super-argus/server-env";
			description = ''
				Secrets, not in git: ARGUS_JWT_SECRET (required), ARGUS_ADMIN_EMAIL
				and ARGUS_ADMIN_PASSWORD (seed the first admin user; harmless to
				leave set afterwards, SeedAdmin only acts when no users exist yet).
				  sudo install -Dm 600 /dev/stdin /etc/super-argus/server-env
			'';
		};

		agent = {
			enable = lib.mkOption {
				type = lib.types.bool;
				default = true;
				description = "Whether to also run argus-agent, shipping this host's own journal to the server above.";
			};

			mode = lib.mkOption {
				type = lib.types.str;
				default = "auto";
				description = "ARGUS_AGENT_MODE: \"auto\", \"journal\", or \"kubernetes\".";
			};

			actions = {
				enable = lib.mkOption {
					type = lib.types.bool;
					default = false;
					description = ''
						Let admins restart the units listed in `units` from the Argus UI/API.
						The agent polls the server for requests (it opens no port) and a
						polkit rule lets only its dynamic user restart exactly those units.
					'';
				};

				units = lib.mkOption {
					type = lib.types.listOf lib.types.str;
					default = [ ];
					example = [ "nginx.service" ];
					description = "Full unit names (with .service) the agent may restart. Empty means nothing is restartable.";
				};
			};

			environmentFile = lib.mkOption {
				type = lib.types.str;
				default = "/etc/super-argus/agent-env";
				description = ''
					ARGUS_URL and ARGUS_API_KEY for the agent. The API key can only be
					minted from the running server (as admin), so this file won't exist
					on first boot; argus-agent.service is conditioned on it and simply
					doesn't start until it's written:
					  sudo install -Dm 600 /dev/stdin /etc/super-argus/agent-env
					  sudo systemctl start argus-agent
				'';
			};
		};
	};

	config = lib.mkIf cfg.enable {
		users.users.super-argus = {
			isSystemUser = true;
			group = "super-argus";
			home = stateDir;
		};
		users.groups.super-argus = { };

		# Lets an operator run `argus tui` (or `process-compose attach`) from a
		# login shell on this host without `nix develop`/`nix shell` first.
		environment.systemPackages = [ pkgs.process-compose ];

		# binDir is 0755 root:root specifically so any user can exec `argus`
		# out of it, but that's moot if stateDir itself blocks traversal --
		# 0750 super-argus:super-argus did exactly that. 0755 here only grants
		# *traversal* into stateDir; dataDir keeps its own 0700 lockdown
		# underneath, so this doesn't expose server data.
		environment.interactiveShellInit = ''
			export PATH="$PATH:${binDir}"
		'';

		systemd.tmpfiles.rules = [
			"d ${stateDir} 0755 super-argus super-argus -"
			"d ${binDir} 0755 root root -"
			"d ${cfg.dataDir} 0700 super-argus super-argus -"
			"d ${buildDir} 0755 root root -"
			"d ${goCache} 0755 root root -"
		];

		# Skipped (not failed) until the source is actually there.
		system.activationScripts.argusBuild = {
			deps = [ ];
			text = ''
				if [ -d ${argusSrc} ]; then
					export PATH=${pkgs.nodejs_22}/bin:${pkgs.bash}/bin:${pkgs.go}/bin:$PATH
					export HOME=${goCache}
					export GOCACHE=${goCache}/build
					export GOPATH=${goCache}/path
					# glebarez/sqlite is pure Go; no C toolchain needed.
					export CGO_ENABLED=0

					rm -rf ${srcDir}
					mkdir -p ${srcDir}
					cp -r ${argusSrc}/. ${srcDir}
					chmod -R u+w ${srcDir}

					( cd ${srcDir}/app && npm ci && npm run build:web )

					cd ${srcDir}
					go build -buildvcs=false -o ${binDir}/argus.new ./cmd/argus
					mv -f ${binDir}/argus.new ${binDir}/argus

					cd ${srcDir}/agent
					go build -buildvcs=false -o ${binDir}/argus-agent.new .
					mv -f ${binDir}/argus-agent.new ${binDir}/argus-agent

					${pkgs.systemd}/bin/systemctl try-restart argus.service argus-agent.service || true
				else
					echo "super-argus: ${argusSrc} not found, skipping build"
				fi
			'';
		};

		systemd.services.argus = {
			description = "super-argus server";
			after = [ "network-online.target" ];
			wants = [ "network-online.target" ];
			wantedBy = [ "multi-user.target" ];

			unitConfig.ConditionPathExists = "${binDir}/argus";

			environment = {
				ARGUS_HTTP_ADDR = addr;
				ARGUS_DATA_DIR = cfg.dataDir;
			};

			serviceConfig = {
				ExecStart = "${binDir}/argus";
				# "-": optional, so the service still starts (and fails fast on the
				# missing ARGUS_JWT_SECRET) before the file exists.
				EnvironmentFile = "-${cfg.environmentFile}";
				Restart = "always";
				RestartSec = 5;
				User = "super-argus";
				Group = "super-argus";
				NoNewPrivileges = true;
				PrivateTmp = true;
				ProtectSystem = "strict";
				ProtectHome = true;
				ReadWritePaths = [ cfg.dataDir ];
				ProtectKernelTunables = true;
				ProtectKernelModules = true;
				ProtectControlGroups = true;
				RestrictSUIDSGID = true;
				LockPersonality = true;
				RestrictAddressFamilies = [ "AF_INET" "AF_INET6" "AF_UNIX" ];
				UMask = "0077";
			};
		};

		systemd.services.argus-agent = lib.mkIf cfg.agent.enable {
			description = "super-argus agent (ships this host's journal to the local argus server)";
			after = [ "network-online.target" "argus.service" ];
			wants = [ "network-online.target" ];
			wantedBy = [ "multi-user.target" ];

			# No API key until an admin mints one from the running server, so
			# don't crash-loop before that file has been written.
			unitConfig.ConditionPathExists = cfg.agent.environmentFile;

			environment = {
				ARGUS_AGENT_MODE = cfg.agent.mode;
			} // lib.optionalAttrs cfg.agent.actions.enable {
				ARGUS_ACTIONS_ENABLED = "true";
				ARGUS_ACTION_UNITS = lib.concatStringsSep "," cfg.agent.actions.units;
			};

			serviceConfig = {
				ExecStart = "${binDir}/argus-agent";
				EnvironmentFile = cfg.agent.environmentFile;
				Restart = "always";
				RestartSec = 5;
				DynamicUser = true;
				StateDirectory = "argus-agent";
				# systemd-journal to read the journal; super-argus because binDir's
				# parent (stateDir, 0750 super-argus:super-argus) isn't traversable
				# otherwise -- DynamicUser gets its own ephemeral group by default,
				# not this service's static one.
				SupplementaryGroups = [ "systemd-journal" "super-argus" ];
				NoNewPrivileges = true;
				ProtectSystem = "strict";
				ProtectHome = true;
			};
		};

		# The agent runs unprivileged (DynamicUser = argus-agent); this is the only
		# thing that lets it restart anything, and only the listed units.
		security.polkit = lib.mkIf (cfg.agent.enable && cfg.agent.actions.enable) {
			enable = true;
			extraConfig = ''
				polkit.addRule(function(action, subject) {
					if (action.id == "org.freedesktop.systemd1.manage-units" &&
					    subject.user == "argus-agent" &&
					    action.lookup("verb") == "restart" &&
					    ${builtins.toJSON cfg.agent.actions.units}.indexOf(action.lookup("unit")) >= 0) {
						return polkit.Result.YES;
					}
				});
			'';
		};
	};
}
