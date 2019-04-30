from oktabeat import BaseTest

import os


class Test(BaseTest):

    def test_base(self):
        """
        Basic test with exiting Oktabeat normally
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*"
        )

        oktabeat_proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("oktabeat is running"))
        exit_code = oktabeat_proc.kill_and_wait()
        assert exit_code == 0
