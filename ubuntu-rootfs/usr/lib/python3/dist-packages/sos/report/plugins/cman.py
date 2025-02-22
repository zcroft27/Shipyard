# This file is part of the sos project: https://github.com/sosreport/sos
#
# This copyrighted material is made available to anyone wishing to use,
# modify, copy, or redistribute it subject to the terms and conditions of
# version 2 of the GNU General Public License.
#
# See the LICENSE file in the source distribution for further information.

from glob import glob
from sos.report.plugins import Plugin, RedHatPlugin


class Cman(Plugin, RedHatPlugin):

    short_desc = 'cman based Red Hat Cluster High Availability'

    plugin_name = "cman"
    profiles = ("cluster",)

    packages = ("luci", "cman", "clusterlib")

    files = ("/etc/cluster/cluster.conf",)

    def setup(self):

        self.add_copy_spec([
            "/etc/cluster.conf",
            "/etc/cluster",
            "/etc/sysconfig/cluster",
            "/etc/sysconfig/cman",
            "/var/log/cluster",
            "/etc/fence_virt.conf",
            "/var/lib/luci/data/luci.db",
            "/var/lib/luci/etc",
            "/var/log/luci"
        ])

        self.add_cmd_output([
            "cman_tool services",
            "cman_tool nodes",
            "cman_tool status",
            "ccs_tool lsnode",
            "mkqdisk -L",
            "group_tool dump",
            "fence_tool dump",
            "fence_tool ls -n",
            "clustat",
            "rg_test test /etc/cluster/cluster.conf"
        ])

    def postproc(self):
        for cluster_conf in glob("/etc/cluster/cluster.conf*"):
            self.do_file_sub(
                cluster_conf,
                r"(\s*\<fencedevice\s*.*\s*passwd\s*=\s*)\S+(\")",
                r'\1"***"'
            )

        self.do_path_regex_sub(
            r"/var/lib/luci/etc/.*\.ini",
            r"(.*secret\s*=\s*)\S+",
            r"\1******"
        )

# vim: et ts=4 sw=4
