"""
File to run KLC3 on MP1.
Include: 
    - basic tests
        - concrete test
        - symbolic test
    - regression tests
    - subroutine tests
"""

import argparse
import shutil
import subprocess
import os

"""
global variables
"""

EXIT_CODE_SUCCESS = 0
EXIT_CODE_STUDENT_CODE_ERROR = 1
EXIT_CODE_AUTOGRADER_FAILURE = 255

PASS_ICON = '<img src="https://upload.wikimedia.org/wikipedia/commons/thumb/0/03/Green_check.svg/13px-Green_check.svg.png" alt="pass"></img>'
FAIL_ICON = '<img src="https://upload.wikimedia.org/wikipedia/en/thumb/b/ba/Red_x.svg/13px-Red_x.svg.png" alt="fail"></img>'


"""
helper functions
"""

def generate_readme(easy_test_report: str, regression_report: str, report: str, mp3_subroutine_report: str,
                    output_dir: str) -> None:
    content = open("templates/klc3_report.md", "r").read()
    content = content.replace("{{EASY_TEST_REPORT}}", easy_test_report)
    content = content.replace("{{REGRESSION_REPORT}}", regression_report)
    content = content.replace("{{MP3_SUBROUTINE_REPORT}}", mp3_subroutine_report)
    content = content.replace("{{REPORT}}", report)
    open(os.path.join(output_dir, "README.md"), "w").write(content)


"""
main functions
"""

def launcher(directory: str):
    # copy the replay file to this student commit dir
    shutil.copy("templates/replay.sh", directory)
    
    # parse student code
    proc = subprocess.Popen(["klc3-parser", "student.asm"], cwd=directory, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    with open(os.path.join(directory, "klc3-parser-student.log", "w"), "w") as f:
        # poll the process until the process has output
        while proc.poll() is None:
            line = proc.stdout.readline()
            f.write(line.decode("utf-8"))
        exit_code = proc.wait()
        # compilation failed
        if exit_code != 0:
            cprint("klc3-parser on student code failed", "red")
            generate_readme(FAIL_ICON + " Your code failed to compile.\n" \
                        + "\n" \
                        + "lc3as output:\n" \
                        + "```\n" \
                        + open(os.path.join(output_dir, "klc3-parser-student.log"), "r").read() \
                        + "\n```", 
                        FAIL_ICON + " Your code failed to compile.",
                        FAIL_ICON + " Your code failed to compile.",
                        FAIL_ICON + " Your code failed to compile.",
                        directory
                        )
            


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='KLC3 on MP1')
    parser.add_argument('-d', '--dir', type=str)
    argv = parser.parse_args()

    exit(launcher(argv.dir))
