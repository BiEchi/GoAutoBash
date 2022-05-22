import argparse
import subprocess
from termcolor import cprint
from typing import Optional, List
import os
import shutil
import glob
import tempfile

EXIT_CODE_SUCCESS = 0
EXIT_CODE_STUDENT_CODE_ERROR = 1
EXIT_CODE_AUTOGRADER_FAILURE = 255

PASS_ICON = '<img src="https://upload.wikimedia.org/wikipedia/commons/thumb/0/03/Green_check.svg/13px-Green_check.svg.png" alt="pass"></img>'
FAIL_ICON = '<img src="https://upload.wikimedia.org/wikipedia/en/thumb/b/ba/Red_x.svg/13px-Red_x.svg.png" alt="fail"></img>'


def remove_extension(filename: str) -> str:
    return os.path.splitext(filename)[0]


def run_test(student_filename: str, input_data_list: [str], case_start_index: int, concrete_case_name: Optional[str],
             output_dir: str, entry_file: Optional[str] = None, gold_program: str = "mp3_gold",
             additional_options: Optional[list] = None) -> int:
    """
    Run regression/symbolic test of one subroutine on the student code
    :param student_filename: need to be parsed already
    :param input_data_list:
    :param case_start_index:
    :param concrete_case_name: case name for easy tests and regression test, None for symbolic
    :param output_dir:
    :param entry_file:
    :param gold_program:
    :param additional_options:
    :return: 0 for success (may or may not have issues), 1 for student code error. Exit with 255 for workflow failure.
    """

    # Run klc3
    klc3_command = [
        "klc3",
        "--report-to-terminal",
        "--output-dir=%s" % output_dir,
        "--report-style=clang",
        "--test-case-start-index=%d" % case_start_index,
        # MP3 options
        # "--strict-memory-assertion=true",
        "--max-lc3-step-count=200000",
        "--max-lc3-out-length=1100",
        "--single-state-max-time=10000",
        "--early-exit-time=300"  # 5min
    ]
    if additional_options is not None:
        klc3_command += additional_options

    if concrete_case_name is not None:  # easy test or regression test
        klc3_command += [
            "--unique-test-case-name=%s" % concrete_case_name,
            "--output-flowgraph=false",
        ]
    else:  # symbolic test
        klc3_command += [
            "--output-flowgraph"
        ]

    # Shared input file
    klc3_command += [
        "sched_alloc_",
        "stack_alloc_"
    ]
    klc3_command += input_data_list
    # Test programs
    klc3_command += [
        "--test", remove_extension(student_filename),
    ]
    if entry_file is not None:
        klc3_command += ["--test", remove_extension(entry_file)]
    # Gold program
    klc3_command += [
        "--gold", gold_program,
    ]

    proc = subprocess.Popen(klc3_command, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    with open(os.path.join(output_dir, "klc3.log"), "w") as log_file:
        log_file.write("$ %s\n\n" % "\n".join(klc3_command))
        while proc.poll() is None:
            line = proc.stdout.readline()
            log_file.write(line.decode())
            # print(line.decode(), end="")
    exit_code = proc.wait()
    if exit_code != 0:
        cprint("klc3 terminated abnormally", "red")
        exit(EXIT_CODE_AUTOGRADER_FAILURE)

    return EXIT_CODE_SUCCESS


def enum_input_data_list(case_dir: str) -> [str]:
    ret = []
    input_list = glob.glob(os.path.join(case_dir, "*.asm"))
    for input_data_file in input_list:
        basename = remove_extension(input_data_file)
        ret.append(basename)
        if not os.path.exists(basename + ".info"):  # if already parsed, skip, since regression asm will not be changed
            proc = subprocess.Popen(["klc3-parser", input_data_file],
                                    stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
            exit_code = proc.wait()
            if exit_code != 0:
                cprint("Error parsing regression input file %s" % input_data_file, "red")
                exit(EXIT_CODE_AUTOGRADER_FAILURE)
    return ret


def run_concrete_test(student_filename: str, input_data_list: List[str], case_name: str, commit_id: Optional[str],
                      root_output_dir: str, root_temp_dir: str, root_store_dir: str,
                      entry_file: Optional[str] = None, gold_program: str = "mp3_gold",
                      additional_options: Optional[list] = None) -> (bool, str):
    """
    Run a concrete case
    :param student_filename:
    :param input_data_list:
    :param case_name:
    :param commit_id:
    :param root_output_dir:
    :param root_temp_dir:
    :param root_store_dir:
    :param entry_file:
    :param gold_program:
    :param additional_options:
    :return: (pass, report content with combined name)
    """

    # Create dirs
    case_store_dir = os.path.join(root_store_dir, case_name)
    os.makedirs(case_store_dir, exist_ok=True)

    # Output directly to the output path, while the test subdirectory is named differently
    run_test(
        student_filename=student_filename,
        input_data_list=input_data_list,
        case_start_index=0,
        concrete_case_name=case_name,
        output_dir=root_output_dir,
        entry_file=entry_file,
        gold_program=gold_program,
        additional_options=additional_options
    )
    # Unless klc3 breaks, report.md is always generated

    ret_pass = True

    ret_report = '<li>'
    case_name_and_link = ' <a href="' + case_name + '">' + case_name + '</a>'
    if commit_id is not None:
        case_name_and_link += ' from your commit ' + commit_id

    # Report are directly output to root output dir
    case_report = open(os.path.join(root_output_dir, "report.md"), "r").read()
    if case_report == "":
        # No issue
        ret_report += PASS_ICON + case_name_and_link
    else:
        ret_report += FAIL_ICON + case_name_and_link + '\n' \
                      + '<details>\n' \
                      + '<summary>Click to expand the report</summary>\n\n<br>\n\n' \
                      + case_report + '\n\n' \
                      + "</details>\n"
        ret_pass = False
    ret_report += '</li>\n\n'
    os.remove(os.path.join(root_output_dir, "report.md"))

    # Move the regression data file into subdirectory and alter the lcs script
    if os.path.isdir(os.path.join(root_output_dir, case_name)):
        for input_file in input_data_list:
            # Use filenames in list but local copy of file
            shutil.move(os.path.join(root_output_dir, os.path.basename(input_file) + ".asm"),
                        os.path.join(root_output_dir, case_name))
        if entry_file is not None:
            shutil.move(os.path.join(root_output_dir, os.path.basename(entry_file) + ".asm"),
                        os.path.join(root_output_dir, case_name))
        lcs_lines = open(os.path.join(root_output_dir, case_name, case_name + ".lcs"),
                         "r").read().splitlines()
        with open(os.path.join(root_output_dir, case_name, case_name + ".lcs"), "w") as lcs_file:
            for line in lcs_lines:
                if line.startswith("f test"):
                    line = "f %s/%s" % (case_name, line[2:])
                lcs_file.write(line + "\n")
    else:
        os.makedirs(os.path.join(root_output_dir, case_name))
        for input_file in input_data_list:
            # Use filenames in list but local copy of file
            shutil.move(os.path.join(root_output_dir, os.path.basename(input_file) + ".asm"),
                        os.path.join(root_output_dir, case_name))
        if entry_file is not None:
            shutil.move(os.path.join(root_output_dir, os.path.basename(entry_file) + ".asm"),
                        os.path.join(root_output_dir, case_name))

    # Move logs to store
    for log_filename in glob.glob(os.path.join(root_output_dir, "*.log")):
        shutil.move(log_filename, case_store_dir)

    return ret_pass, ret_report


def run_all_regression(student_filename: str, regression_dir: str,
                       root_output_dir: str, root_temp_dir: str, root_store_dir: str) -> (bool, str, int):
    """
    Run all regression test. regression_dir has the structure of commit_id/subroutine-test******.
    :param student_filename:
    :param regression_dir:
    :param root_output_dir:
    :param root_temp_dir:
    :param root_store_dir:
    :return: (all pass, output, regression case max index)
    """

    ret_all_pass = True
    ret_report = ""
    ret_max_index = -1

    # Enumerate commits in regression dir
    for commit_path in glob.glob(os.path.join(regression_dir, "*")):
        commit_id = os.path.basename(commit_path)

        for case_dir in glob.glob(os.path.join(commit_path, "*")):
            case_name = os.path.basename(case_dir)
            # In the storage, the test cases won't have "-regression" suffix, but may have previous regression test
            if case_name.startswith("test0") and not case_name.endswith("-regression"):
                # Is a regression (from current commit) test directory
                test_index = int(case_name[4:])
                ret_max_index = max(ret_max_index, test_index)
                combined_name = 'test' + str(test_index).zfill(6) + '-regression'

                case_pass, case_report = run_concrete_test(
                    student_filename=student_filename,
                    input_data_list=enum_input_data_list(case_dir),
                    case_name=combined_name,
                    commit_id=commit_id,
                    root_output_dir=root_output_dir,
                    root_temp_dir=root_temp_dir,
                    root_store_dir=root_store_dir
                )

                if not case_pass:
                    ret_all_pass = False

                ret_report += case_report

    if ret_report == "":
        ret_report = PASS_ICON + " You have no regression test case yet."
    else:
        ret_report = '<ul>\n' + ret_report + '</ul>\n'
    return ret_all_pass, ret_report, ret_max_index


def run_all_easy_tests(student_filename: str,
                       root_output_dir: str, root_temp_dir: str, root_store_dir: str) -> (bool, str):
    """
    Run all easy test
    :param student_filename:
    :param root_output_dir:
    :param root_temp_dir:
    :param root_store_dir:
    :return: (all pass, output)
    """

    ret_all_pass = True
    ret_report = ""

    for case_dir in glob.glob(os.path.join("easy_tests", "*")):
        case_name = os.path.basename(case_dir)

        case_pass, case_report = run_concrete_test(
            student_filename=student_filename,
            input_data_list=enum_input_data_list(case_dir),
            case_name=case_name,
            commit_id=None,
            root_output_dir=root_output_dir,
            root_temp_dir=root_temp_dir,
            root_store_dir=root_store_dir
        )

        if not case_pass:
            ret_all_pass = False

        ret_report += case_report

    return ret_all_pass, '<ul>\n' + ret_report + '</ul>\n'


def run_symbolic(student_filename: str, start_index: int, root_output_dir: str, root_temp_dir: str,
                 root_store_dir: str) -> str:
    """
    Run symbolic test.
    :param student_filename:
    :param start_index:
    :param root_output_dir:
    :param root_temp_dir:
    :param root_store_dir:
    :return:
    """

    ret_report = ""

    run_test(
        student_filename=student_filename,
        input_data_list=[
            "symbolic_sched_.asm",
            "symbolic_extra_.asm",
        ],
        case_start_index=start_index,
        concrete_case_name=None,
        output_dir=root_output_dir,
    )

    report_content = open(os.path.join(root_output_dir, "report.md"), "r").read()
    if report_content == "":
        report_content = PASS_ICON + " KLC3 feedback tool find no issue in your code for now.\n"
    ret_report += report_content + "\n"
    os.remove(os.path.join(root_output_dir, "report.md"))

    # Copy tests to store
    for test_case_dir in glob.glob(os.path.join(root_output_dir, "test0*")):
        if not test_case_dir.endswith("-regression"):
            shutil.copytree(test_case_dir,
                            os.path.join(root_store_dir, os.path.basename(test_case_dir)))
            # In storage, test case doesn't have "-regression" suffix

    # Move logs to store
    for log_filename in glob.glob(os.path.join(root_output_dir, "*.log")):
        shutil.move(log_filename, root_store_dir)

    return ret_report


def run_mp3_subroutine_test(student_filename: str, root_output_dir: str, root_temp_dir: str,
                            root_store_dir: str) -> (bool, str):
    """
    Run MP3 subroutine unit test
    :param student_filename:
    :param root_output_dir:
    :param root_temp_dir:
    :param root_store_dir:
    :return: (pass, report content with combined name)
    """

    # Read sym file of student code to get subroutine addr
    test_subroutine_addr = -1
    targeting_prefix = "//	MP3 "
    for line in open(remove_extension(student_filename) + ".sym", "r"):
        if line.startswith(targeting_prefix):
            test_subroutine_addr = int(line[len(targeting_prefix):].strip(), 16)
            break
    if test_subroutine_addr == -1:
        return False, FAIL_ICON + " Failed to find subroutine MP3 in your code."

    # Generate entry asm
    entry_filename = "test_entry.asm"
    entry_content = open("mp3_subroutine/test_entry.asm", "r").read()
    entry_content = entry_content.replace("{{MP3}}", "x%04X" % (test_subroutine_addr - (0x2FF0 + 1)))
    temp_entry_filename = os.path.join(root_temp_dir, entry_filename)
    open(temp_entry_filename, "w").write(entry_content)

    # Compile temp entry file
    proc = subprocess.Popen(["klc3-parser", temp_entry_filename], stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    with open(os.path.join(root_output_dir, "klc3-parser-entry.log"), "w") as log_file:
        # Will be moved to stored by run_concrete_test
        while proc.poll() is None:
            line = proc.stdout.readline()
            log_file.write(line.decode())
            # spinner.next()
    exit_code = proc.wait()
    if exit_code != 0:
        cprint("klc3-parser on entry code terminated abnormally", "red")
        exit(EXIT_CODE_AUTOGRADER_FAILURE)

    ret_pass, ret_report = run_concrete_test(
        student_filename=student_filename,
        input_data_list=[
            "mp3_subroutine/test_sched_input",
            "mp3_subroutine/test_sched_mem",
            "mp3_subroutine/test_extra_input"
        ],
        entry_file=remove_extension(temp_entry_filename),
        gold_program="mp3_subroutine/gold",
        additional_options=[
            "--cross-compare-r0=true",
            "--cross-compare-output=false",
            "--post-checker=mp1",  # reuse MP1 post checker to check the return location
        ],
        case_name="mp3_subroutine",
        commit_id=None,
        root_output_dir=root_output_dir,
        root_temp_dir=root_temp_dir,
        root_store_dir=root_store_dir
    )

    return ret_pass, '<ul>\n' + ret_report + '</ul>\n'


def generate_readme(easy_test_report: str, regression_report: str, report: str, mp3_subroutine_report: str,
                    output_dir: str) -> None:
    content  open("templates/klc3_report.md", "r").read()
    content = content.replace("EASY_TEST_REPORT", easy_test_report)
    content = content.replace("REGRESSION_REPORT", regression_report)
    content = content.replace("MP3_SUBROUTINE_REPORT", mp3_subroutine_report)
    content = content.replace("REPORT", report)
    open(os.path.join(output_dir, "README.md"), "w").write(content)


def parse_student_code(student_filename: str, output_dir: str) -> str:
    proc = subprocess.Popen(["klc3-parser", student_filename], stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    with open(os.path.join(output_dir, "klc3-parser-student.log"), "w") as log_file:
        while proc.poll() is None:
            line = proc.stdout.readline()
            log_file.write(line.decode())
    exit_code = proc.wait()
    if exit_code != 0:
        cprint("klc3-parser on student code failed", "red")
        return FAIL_ICON + " Your code failed to compile.\n" \
               + "\n" \
               + "lc3as output:\n" \
               + "```\n" \
               + open(os.path.join(output_dir, "klc3-parser-student.log"), "r").read() \
               + "\n```"

    return ""


def run_mp3_test(test_asm: str, regression_dir: str, output_dir: str) -> int:
    # Create root output dir
    os.makedirs(output_dir, exist_ok=True)

    # Create root store dir
    store_dir = os.path.join(output_dir, "store")
    os.makedirs(store_dir, exist_ok=True)

    # Create root temp dir
    temp_dir = tempfile.mkdtemp()

    # Copy files
    shutil.copy("templates/.gitignore", output_dir)
    shutil.copy("templates/replay.sh", output_dir)

    # Copy student code to temp dir
    test_basename = os.path.basename(test_asm)
    temp_student_filename = os.path.join(temp_dir, test_basename)
    shutil.copy(test_asm, temp_student_filename)

    # Parse student code
    parse_report = parse_student_code(temp_student_filename, store_dir)
    if parse_report != "":
        generate_readme(parse_report,
                        FAIL_ICON + " Your code failed to compile.",
                        FAIL_ICON + " Your code failed to compile.",
                        FAIL_ICON + " Your code failed to compile.",
                        output_dir)
        os.system("rm -rf " + temp_dir)
        return EXIT_CODE_SUCCESS

    # Run easy test
    easy_test_all_pass, easy_test_report = run_all_easy_tests(
        student_filename=temp_student_filename,
        root_output_dir=output_dir,
        root_temp_dir=temp_dir,
        root_store_dir=store_dir
    )
    if not easy_test_all_pass:
        generate_readme(easy_test_report,
                        FAIL_ICON + " You have to pass easy test first.",
                        FAIL_ICON + " You have to pass easy test and regression test first.",
                        FAIL_ICON + " You have to pass easy test and regression test first.",
                        output_dir)
        os.system("rm -rf " + temp_dir)
        return EXIT_CODE_SUCCESS

    # Run regression test
    regression_all_pass, regression_report, regression_max_index = run_all_regression(
        student_filename=temp_student_filename,
        regression_dir=regression_dir,
        root_output_dir=output_dir,
        root_temp_dir=temp_dir,
        root_store_dir=store_dir
    )
    if not regression_all_pass:
        generate_readme(easy_test_report,
                        regression_report,
                        FAIL_ICON + " You have to pass easy test and regression test first.",
                        FAIL_ICON + " You have to pass easy test and regression test first.",
                        output_dir)
        os.system("rm -rf " + temp_dir)
        return EXIT_CODE_SUCCESS

    # Run symbolic test
    symbolic_report = run_symbolic(
        student_filename=temp_student_filename,
        start_index=regression_max_index + 1,
        root_output_dir=output_dir,
        root_temp_dir=temp_dir,
        root_store_dir=store_dir,
    )

    # Run MP3 subroutine unit test
    _, mp3_subroutine_report = run_mp3_subroutine_test(
        student_filename=temp_student_filename,
        root_output_dir=output_dir,
        root_temp_dir=temp_dir,
        root_store_dir=store_dir,
    )
    generate_readme(easy_test_report, regression_report, symbolic_report, mp3_subroutine_report, output_dir)
    os.system("rm -rf " + temp_dir)
    return EXIT_CODE_SUCCESS


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument("--regression-dir", help="Regression directory", dest="regression_dir")
    parser.add_argument("--output-dir", help="Output directory", dest="output_dir")
    parser.add_argument("file", help="Student asm file (*.asm)")
    argv = parser.parse_args()

    exit(run_mp3_test(
        test_asm=argv.file,
        regression_dir=argv.regression_dir,
        output_dir=argv.output_dir,
    ))
