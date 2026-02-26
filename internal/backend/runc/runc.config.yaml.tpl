{
  "ociVersion": "1.0.2",
  "process": {
    "terminal": false,
    "user": {
      "uid": 0,
      "gid": 0
    },

    "capabilities": {
      "bounding": [
        "CAP_NET_RAW",
        "CAP_NET_ADMIN"
      ],
      "effective": [
        "CAP_NET_RAW",
        "CAP_NET_ADMIN"
      ],
      "permitted": [
        "CAP_NET_RAW",
        "CAP_NET_ADMIN"
      ],
      "ambient": [
        "CAP_NET_RAW",
        "CAP_NET_ADMIN"
      ]
    },

    "args": [
      {{- range $i, $arg := .BuildArgs }}
      {{- if $i}},{{ end }}"{{ $arg }}"
      {{- end }}
    ],
    "cwd": "/",
    "env": [
      "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
      "TERM=xterm",
      "HOST_TYPE={{ .HostType }}",
      "HOST_NAME={{ .HostName }}",
      "HOST_DIR=/host"
      {{- range .Env }},
      {{ . | jsonQuote }}
      {{- end }}
    ]
  },

  "root": {
    "path": "{{ .RootFs }}",
    "readonly": false
  },

  "annotations": {
    "plugin.id": "{{ .PluginId }}",
    "plugin.version": "{{ .PluginVersion }}",
    "device.id": "{{ .DeviceId }}"
  },

  "mounts": [
    {
      "destination": "/host",
      "type": "bind",
      "source": "/",
      "options": [
        "rbind",
        "ro",
        "rslave"
      ]
    },
    {
      "destination": "/proc",
      "type": "proc",
      "source": "proc",
      "options": ["nosuid", "noexec", "nodev"]
    },
    {
      "destination": "/dev",
      "type": "tmpfs",
      "source": "tmpfs",
      "options": ["nosuid", "strictatime", "mode=755", "size=65536k"]
    },
    {
      "destination": "/dev/pts",
      "type": "devpts",
      "source": "devpts",
      "options": [
        "nosuid",
        "noexec",
        "newinstance",
        "ptmxmode=0666",
        "mode=0620",
        "gid=5"
      ]
    },
    {
      "destination": "/sys",
      "type": "sysfs",
      "source": "sysfs",
      "options": ["nosuid", "noexec", "nodev", "ro"]
    },
    {
      "destination": "/sys/fs/cgroup",
      "type": "cgroup",
      "source": "cgroup",
      "options": ["ro", "nosuid", "noexec", "nodev"]
    },
    {
      "destination": "/etc/hosts",
      "type": "bind",
      "source": "/etc/hosts",
      "options": ["rbind", "ro"]
    },
    {
      "destination": "/etc/resolv.conf",
      "type": "bind",
      "source": "/etc/resolv.conf",
      "options": ["rbind", "ro"]
    },
    {
      "destination": "/opt/edge-agent/runtime",
      "type": "bind",
      "source": "/opt/edge-agent/runtime",
      "options": ["rbind", "rw"]
    }
  ],

  "linux": {
    "seccomp": {
      "defaultAction": "SCMP_ACT_ALLOW"
    },

    "resources": {
      "cpu": {
        "shares": {{ .CPU }},
        "period": 100000
      },
      "memory": {
        "limit": {{ .Memory }}
      }
    },

    "namespaces": [
      { "type": "pid" },
      { "type": "mount" },
      { "type": "ipc" },
      { "type": "uts" }
    ]
  }
}