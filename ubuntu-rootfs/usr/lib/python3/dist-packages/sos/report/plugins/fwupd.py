# This file is part of the sos project: https://github.com/sosreport/sos
#
# This copyrighted material is made available to anyone wishing to use,
# modify, copy, or redistribute it subject to the terms and conditions of
# version 2 of the GNU General Public License.
#
# See the LICENSE file in the source distribution for further information.

from sos.report.plugins import Plugin, IndependentPlugin, SoSPredicate


class Fwupd(Plugin, IndependentPlugin):

    short_desc = 'fwupd information'

    plugin_name = 'fwupd'
    profiles = ('system', )
    services = ('fwupd',)
    packages = ('fwupd',)

    def setup(self):
        self.set_cmd_predicate(SoSPredicate(self, services=["fwupd"]))
        self.add_cmd_output([
            "fwupdmgr get-approved-firmware",
            "fwupdmgr get-devices --no-unreported-check",
            "fwupdmgr get-history",
            "fwupdmgr get-remotes",
            # collect json format using fwupdagent
            "/usr/libexec/fwupd/fwupdagent get-devices",
            "/usr/libexec/fwupd/fwupdagent get-updates",
        ])

        self.add_copy_spec("/etc/fwupd")

    def postproc(self):
        self.do_path_regex_sub(
            "/etc/fwupd/remotes.d/*",
            r"(Password)=(.*)",
            r"\1=********"
        )

        self.do_file_sub(
            "/etc/fwupd/redfish.conf",
            r"(Password)=(.*)",
            r"\1=********"
        )


# vim: et ts=4 sw=4
