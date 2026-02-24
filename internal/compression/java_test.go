package compression

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// JavaCompressor metadata tests
// ---------------------------------------------------------------------------

func TestJavaCompressor_Language(t *testing.T) {
	c := NewJavaCompressor()
	assert.Equal(t, "java", c.Language())
}

func TestJavaCompressor_SupportedNodeTypes(t *testing.T) {
	c := NewJavaCompressor()
	types := c.SupportedNodeTypes()
	assert.Contains(t, types, "package_declaration")
	assert.Contains(t, types, "import_declaration")
	assert.Contains(t, types, "class_declaration")
	assert.Contains(t, types, "interface_declaration")
	assert.Contains(t, types, "method_declaration")
	assert.Contains(t, types, "constructor_declaration")
	assert.Contains(t, types, "enum_declaration")
	assert.Contains(t, types, "annotation_type_declaration")
	assert.Contains(t, types, "record_declaration")
}

func TestJavaCompressor_EmptyInput(t *testing.T) {
	c := NewJavaCompressor()
	output, err := c.Compress(context.Background(), []byte{})
	require.NoError(t, err)
	assert.Empty(t, output.Signatures)
	assert.Equal(t, 0, output.OriginalSize)
	assert.Equal(t, "java", output.Language)
}

func TestJavaCompressor_ContextCancellation(t *testing.T) {
	c := NewJavaCompressor()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Generate large source to ensure cancellation is checked.
	var b strings.Builder
	for i := 0; i < 5000; i++ {
		b.WriteString("// line\n")
	}

	_, err := c.Compress(ctx, []byte(b.String()))
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// Package declaration
// ---------------------------------------------------------------------------

func TestJavaCompressor_PackageDeclaration(t *testing.T) {
	c := NewJavaCompressor()
	source := `package com.example.app;`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindImport, sig.Kind)
	assert.Equal(t, "com.example.app", sig.Name)
	assert.Contains(t, sig.Source, "package com.example.app;")
}

// ---------------------------------------------------------------------------
// Import declarations
// ---------------------------------------------------------------------------

func TestJavaCompressor_ImportDeclarations(t *testing.T) {
	c := NewJavaCompressor()
	source := `import java.util.List;
import java.util.Map;
import static java.lang.Math.PI;`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	importCount := 0
	for _, sig := range output.Signatures {
		if sig.Kind == KindImport {
			importCount++
		}
	}
	assert.Equal(t, 3, importCount)
}

func TestJavaCompressor_WildcardImport(t *testing.T) {
	c := NewJavaCompressor()
	source := `import java.util.*;`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	assert.Equal(t, KindImport, output.Signatures[0].Kind)
	assert.Contains(t, output.Signatures[0].Source, "import java.util.*;")
}

// ---------------------------------------------------------------------------
// Class declarations
// ---------------------------------------------------------------------------

func TestJavaCompressor_SimpleClass(t *testing.T) {
	c := NewJavaCompressor()
	source := `public class Foo {
    private int x;
    public void bar() {
        System.out.println("hello");
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig, "expected a class signature")
	assert.Equal(t, "Foo", classSig.Name)
	assert.Contains(t, classSig.Source, "public class Foo {")
	assert.Contains(t, classSig.Source, "private int x;")
	assert.Contains(t, classSig.Source, "public void bar()")
	assert.NotContains(t, classSig.Source, "System.out.println")
	assert.Contains(t, classSig.Source, "}")
}

func TestJavaCompressor_ClassExtendsImplements(t *testing.T) {
	c := NewJavaCompressor()
	source := `public class MyService extends BaseService implements Runnable, Serializable {
    @Override
    public void run() {
        doWork();
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig)
	assert.Equal(t, "MyService", classSig.Name)
	assert.Contains(t, classSig.Source, "extends BaseService")
	assert.Contains(t, classSig.Source, "implements Runnable, Serializable")
	assert.NotContains(t, classSig.Source, "doWork()")
}

func TestJavaCompressor_ClassWithConstructor(t *testing.T) {
	c := NewJavaCompressor()
	source := `public class Config {
    private final String name;

    public Config(String name) {
        this.name = name;
    }

    public String getName() {
        return name;
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig)
	assert.Equal(t, "Config", classSig.Name)
	assert.Contains(t, classSig.Source, "private final String name;")
	assert.Contains(t, classSig.Source, "public Config(String name)")
	assert.Contains(t, classSig.Source, "public String getName()")
	assert.NotContains(t, classSig.Source, "this.name = name")
	assert.NotContains(t, classSig.Source, "return name")
}

func TestJavaCompressor_AbstractClass(t *testing.T) {
	c := NewJavaCompressor()
	source := `public abstract class Shape {
    abstract double area();
    abstract double perimeter();
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig)
	assert.Equal(t, "Shape", classSig.Name)
	assert.Contains(t, classSig.Source, "abstract double area();")
	assert.Contains(t, classSig.Source, "abstract double perimeter();")
}

// ---------------------------------------------------------------------------
// Interface declarations
// ---------------------------------------------------------------------------

func TestJavaCompressor_Interface(t *testing.T) {
	c := NewJavaCompressor()
	source := `public interface Repository<T> {
    T findById(long id);
    List<T> findAll();
    void save(T entity);
    void delete(long id);
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var ifaceSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindInterface {
			ifaceSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, ifaceSig, "expected an interface signature")
	assert.Equal(t, "Repository", ifaceSig.Name)
	assert.Contains(t, ifaceSig.Source, "public interface Repository<T> {")
	assert.Contains(t, ifaceSig.Source, "T findById(long id);")
	assert.Contains(t, ifaceSig.Source, "List<T> findAll();")
	assert.Contains(t, ifaceSig.Source, "void save(T entity);")
	assert.Contains(t, ifaceSig.Source, "void delete(long id);")
}

func TestJavaCompressor_InterfaceWithDefaultMethod(t *testing.T) {
	c := NewJavaCompressor()
	source := `public interface Greeter {
    String greet(String name);
    default String greetAll(List<String> names) {
        return names.stream().map(this::greet).collect(Collectors.joining(", "));
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var ifaceSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindInterface {
			ifaceSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, ifaceSig)
	assert.Equal(t, "Greeter", ifaceSig.Name)
	assert.Contains(t, ifaceSig.Source, "String greet(String name);")
	assert.Contains(t, ifaceSig.Source, "default String greetAll(List<String> names)")
	assert.NotContains(t, ifaceSig.Source, "Collectors.joining")
}

func TestJavaCompressor_InterfaceExtends(t *testing.T) {
	c := NewJavaCompressor()
	source := `public interface CrudRepository<T> extends Repository<T> {
    void update(T entity);
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var ifaceSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindInterface {
			ifaceSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, ifaceSig)
	assert.Equal(t, "CrudRepository", ifaceSig.Name)
	assert.Contains(t, ifaceSig.Source, "extends Repository<T>")
}

// ---------------------------------------------------------------------------
// Enum declarations
// ---------------------------------------------------------------------------

func TestJavaCompressor_SimpleEnum(t *testing.T) {
	c := NewJavaCompressor()
	source := `public enum Color {
    RED,
    GREEN,
    BLUE
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var enumSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindType {
			enumSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, enumSig, "expected an enum signature")
	assert.Equal(t, "Color", enumSig.Name)
	assert.Contains(t, enumSig.Source, "public enum Color {")
	assert.Contains(t, enumSig.Source, "RED")
	assert.Contains(t, enumSig.Source, "GREEN")
	assert.Contains(t, enumSig.Source, "BLUE")
}

func TestJavaCompressor_EnumWithMethods(t *testing.T) {
	c := NewJavaCompressor()
	source := `public enum Planet {
    MERCURY(3.303e+23, 2.4397e6),
    VENUS(4.869e+24, 6.0518e6);

    private final double mass;
    private final double radius;

    Planet(double mass, double radius) {
        this.mass = mass;
        this.radius = radius;
    }

    double surfaceGravity() {
        return G * mass / (radius * radius);
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var enumSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindType {
			enumSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, enumSig)
	assert.Equal(t, "Planet", enumSig.Name)
	assert.Contains(t, enumSig.Source, "MERCURY")
	assert.Contains(t, enumSig.Source, "VENUS")
}

// ---------------------------------------------------------------------------
// Annotation type declarations
// ---------------------------------------------------------------------------

func TestJavaCompressor_AnnotationType(t *testing.T) {
	c := NewJavaCompressor()
	source := `public @interface MyAnnotation {
    String value();
    int count() default 0;
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var annotSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindType && output.Signatures[i].Name == "MyAnnotation" {
			annotSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, annotSig, "expected an annotation type signature")
	assert.Equal(t, "MyAnnotation", annotSig.Name)
	assert.Contains(t, annotSig.Source, "@interface MyAnnotation {")
	assert.Contains(t, annotSig.Source, "String value();")
	assert.Contains(t, annotSig.Source, "int count() default 0;")
}

// ---------------------------------------------------------------------------
// Record declarations (Java 16+)
// ---------------------------------------------------------------------------

func TestJavaCompressor_RecordDeclaration(t *testing.T) {
	c := NewJavaCompressor()
	source := `public record Point(int x, int y) {
    public double distanceFromOrigin() {
        return Math.sqrt(x * x + y * y);
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var recordSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			recordSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, recordSig, "expected a record signature")
	assert.Equal(t, "Point", recordSig.Name)
	assert.Contains(t, recordSig.Source, "record Point(int x, int y)")
	assert.Contains(t, recordSig.Source, "public double distanceFromOrigin()")
	assert.NotContains(t, recordSig.Source, "Math.sqrt")
}

func TestJavaCompressor_SimpleRecord(t *testing.T) {
	c := NewJavaCompressor()
	source := `public record Pair<A, B>(A first, B second) {}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var recordSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			recordSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, recordSig)
	assert.Equal(t, "Pair", recordSig.Name)
}

// ---------------------------------------------------------------------------
// Javadoc comments
// ---------------------------------------------------------------------------

func TestJavaCompressor_JavadocOnClass(t *testing.T) {
	c := NewJavaCompressor()
	source := `/**
 * A service for managing users.
 */
public class UserService {
    public void createUser(String name) {
        // create user
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig)
	assert.Contains(t, classSig.Source, "A service for managing users")
	assert.Contains(t, classSig.Source, "public class UserService {")
}

func TestJavaCompressor_SingleLineJavadoc(t *testing.T) {
	c := NewJavaCompressor()
	source := `/** Simple helper class. */
public class Helper {
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig)
	assert.Contains(t, classSig.Source, "Simple helper class")
	assert.Contains(t, classSig.Source, "public class Helper {")
}

// ---------------------------------------------------------------------------
// Annotations on declarations
// ---------------------------------------------------------------------------

func TestJavaCompressor_AnnotationsOnClass(t *testing.T) {
	c := NewJavaCompressor()
	source := `@Entity
@Table(name = "users")
public class User {
    @Id
    private Long id;

    @Column(name = "name")
    private String name;
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig)
	assert.Equal(t, "User", classSig.Name)
	assert.Contains(t, classSig.Source, "@Entity")
	assert.Contains(t, classSig.Source, `@Table(name = "users")`)
	assert.Contains(t, classSig.Source, "public class User {")
	// Field annotations inside class body should be captured.
	assert.Contains(t, classSig.Source, "@Id")
	assert.Contains(t, classSig.Source, "@Column")
}

func TestJavaCompressor_OverrideAnnotation(t *testing.T) {
	c := NewJavaCompressor()
	source := `public class Dog extends Animal {
    @Override
    public String speak() {
        return "woof";
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig)
	assert.Contains(t, classSig.Source, "@Override")
	assert.Contains(t, classSig.Source, "public String speak()")
	assert.NotContains(t, classSig.Source, "woof")
}

// ---------------------------------------------------------------------------
// Method declarations with throws
// ---------------------------------------------------------------------------

func TestJavaCompressor_MethodWithThrows(t *testing.T) {
	c := NewJavaCompressor()
	source := `public class FileProcessor {
    public void process(String path) throws IOException, ParseException {
        // processing logic
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig)
	assert.Contains(t, classSig.Source, "throws IOException, ParseException")
	assert.NotContains(t, classSig.Source, "processing logic")
}

// ---------------------------------------------------------------------------
// Nested class
// ---------------------------------------------------------------------------

func TestJavaCompressor_NestedClass(t *testing.T) {
	c := NewJavaCompressor()
	source := `public class Outer {
    private int x;

    public static class Inner {
        private int y;
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig)
	assert.Equal(t, "Outer", classSig.Name)
	assert.Contains(t, classSig.Source, "private int x;")
	assert.Contains(t, classSig.Source, "public static class Inner")
}

// ---------------------------------------------------------------------------
// Compression ratio
// ---------------------------------------------------------------------------

func TestJavaCompressor_CompressionRatio(t *testing.T) {
	c := NewJavaCompressor()
	source := `package com.example;

import java.util.List;
import java.util.ArrayList;

/**
 * Main application class.
 */
public class Application {
    private final List<String> items;

    public Application() {
        this.items = new ArrayList<>();
    }

    public void addItem(String item) {
        if (item == null) {
            throw new IllegalArgumentException("item cannot be null");
        }
        items.add(item);
        System.out.println("Added: " + item);
    }

    public List<String> getItems() {
        return new ArrayList<>(items);
    }

    public int getCount() {
        return items.size();
    }

    public static void main(String[] args) {
        Application app = new Application();
        app.addItem("first");
        app.addItem("second");
        System.out.println("Count: " + app.getCount());
        for (String item : app.getItems()) {
            System.out.println("  - " + item);
        }
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	rendered := output.Render()
	ratio := float64(len(rendered)) / float64(len(source))
	// Expect 40-60% compression (ratio between 0.4 and 0.6).
	assert.Greater(t, ratio, 0.2, "compression ratio too low (too much removed)")
	assert.Less(t, ratio, 0.75, "compression ratio too high (not enough removed)")
}

// ---------------------------------------------------------------------------
// Spring Boot controller fixture
// ---------------------------------------------------------------------------

func TestJavaCompressor_SpringBootController(t *testing.T) {
	c := NewJavaCompressor()
	source := `package com.example.web;

import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.PathVariable;
import java.util.List;

/**
 * REST controller for managing users.
 */
@RestController
public class UserController {

    private final UserService userService;

    public UserController(UserService userService) {
        this.userService = userService;
    }

    @GetMapping("/users")
    public List<User> listUsers() {
        return userService.findAll();
    }

    @GetMapping("/users/{id}")
    public User getUser(@PathVariable Long id) {
        return userService.findById(id)
            .orElseThrow(() -> new NotFoundException("User not found: " + id));
    }

    @PostMapping("/users")
    public User createUser(@RequestBody CreateUserRequest request) {
        User user = new User();
        user.setName(request.getName());
        user.setEmail(request.getEmail());
        return userService.save(user);
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	// Should have package, imports, and class.
	hasPackage := false
	importCount := 0
	hasClass := false
	for _, sig := range output.Signatures {
		switch sig.Kind {
		case KindImport:
			if sig.Name == "com.example.web" {
				hasPackage = true
			} else {
				importCount++
			}
		case KindClass:
			hasClass = true
			assert.Equal(t, "UserController", sig.Name)
			assert.Contains(t, sig.Source, "@RestController")
			assert.Contains(t, sig.Source, "UserService userService")
			assert.Contains(t, sig.Source, "@GetMapping")
			assert.Contains(t, sig.Source, "List<User> listUsers()")
			assert.Contains(t, sig.Source, "User getUser(@PathVariable Long id)")
			assert.Contains(t, sig.Source, "User createUser(@RequestBody CreateUserRequest request)")
			assert.NotContains(t, sig.Source, "orElseThrow")
			assert.NotContains(t, sig.Source, "user.setName")
		}
	}
	assert.True(t, hasPackage, "expected package declaration")
	assert.Equal(t, 6, importCount, "expected 6 imports")
	assert.True(t, hasClass, "expected class declaration")
}

// ---------------------------------------------------------------------------
// Generic interface fixture
// ---------------------------------------------------------------------------

func TestJavaCompressor_GenericInterface(t *testing.T) {
	c := NewJavaCompressor()
	source := `package com.example.repo;

/**
 * Generic repository interface with type parameters.
 *
 * @param <T> the entity type
 * @param <ID> the ID type
 */
public interface GenericRepository<T, ID> {
    T findById(ID id);
    List<T> findAll();
    T save(T entity);
    void deleteById(ID id);
    long count();
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var ifaceSig *Signature
	for _, sig := range output.Signatures {
		if sig.Kind == KindInterface {
			ifaceSig = &sig
			break
		}
	}
	require.NotNil(t, ifaceSig)
	assert.Equal(t, "GenericRepository", ifaceSig.Name)
	assert.Contains(t, ifaceSig.Source, "GenericRepository<T, ID>")
	assert.Contains(t, ifaceSig.Source, "@param <T>")
	assert.Contains(t, ifaceSig.Source, "T findById(ID id);")
	assert.Contains(t, ifaceSig.Source, "long count();")
}

// ---------------------------------------------------------------------------
// Brace counting
// ---------------------------------------------------------------------------

func TestJavaCountBraces(t *testing.T) {
	tests := []struct {
		name string
		line string
		want int
	}{
		{name: "opening brace", line: "public class Foo {", want: 1},
		{name: "closing brace", line: "}", want: -1},
		{name: "balanced", line: "if (x) { y(); }", want: 0},
		{name: "brace in string", line: `String s = "{";`, want: 0},
		{name: "brace in char", line: `char c = '{';`, want: 0},
		{name: "brace after comment", line: `// class Foo {`, want: 0},
		{name: "brace in block comment", line: `/* { */`, want: 0},
		{name: "no braces", line: "int x = 5;", want: 0},
		{name: "nested braces", line: "new HashMap<>() {{ put(1, 2); }}", want: 0},
		{name: "escaped char", line: `char c = '\\';`, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := javaCountBraces(tt.line)
			assert.Equal(t, tt.want, got, "line: %q", tt.line)
		})
	}
}

// ---------------------------------------------------------------------------
// Java modifier stripping
// ---------------------------------------------------------------------------

func TestStripJavaModifiers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "public class", input: "public class Foo", want: "class Foo"},
		{name: "private static final", input: "private static final int x", want: "int x"},
		{name: "abstract class", input: "public abstract class Shape", want: "class Shape"},
		{name: "no modifiers", input: "class Foo", want: "class Foo"},
		{name: "sealed", input: "public sealed class Shape", want: "class Shape"},
		{name: "non-sealed", input: "non-sealed class Circle", want: "class Circle"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripJavaModifiers(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Name extraction
// ---------------------------------------------------------------------------

func TestExtractJavaClassName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "simple", input: "public class Foo {", want: "Foo"},
		{name: "extends", input: "public class Bar extends Baz {", want: "Bar"},
		{name: "generic", input: "public class List<T> {", want: "List"},
		{name: "implements", input: "class Impl implements Iface {", want: "Impl"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJavaClassName(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractJavaInterfaceName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "simple", input: "public interface Foo {", want: "Foo"},
		{name: "generic", input: "interface Repo<T> {", want: "Repo"},
		{name: "extends", input: "interface Sub extends Base {", want: "Sub"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJavaInterfaceName(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractJavaEnumName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "simple", input: "public enum Color {", want: "Color"},
		{name: "implements", input: "enum Status implements Serializable {", want: "Status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJavaEnumName(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Declaration detection
// ---------------------------------------------------------------------------

func TestIsJavaClassDecl(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "public class", input: "public class Foo {", want: true},
		{name: "abstract class", input: "public abstract class Foo {", want: true},
		{name: "final class", input: "final class Foo {", want: true},
		{name: "interface", input: "public interface Foo {", want: false},
		{name: "enum", input: "public enum Foo {", want: false},
		{name: "import", input: "import java.util.class;", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isJavaClassDecl(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsJavaInterfaceDecl(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "public interface", input: "public interface Foo {", want: true},
		{name: "plain interface", input: "interface Foo {", want: true},
		{name: "class", input: "public class Foo {", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isJavaInterfaceDecl(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsJavaRecordDecl(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "public record", input: "public record Point(int x, int y) {", want: true},
		{name: "plain record", input: "record Pair(String a, String b) {", want: true},
		{name: "class", input: "public class Foo {", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isJavaRecordDecl(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Full file integration test
// ---------------------------------------------------------------------------

func TestJavaCompressor_FullFile(t *testing.T) {
	c := NewJavaCompressor()
	source := `package com.example;

import java.util.List;
import java.util.Map;
import java.util.Optional;

/**
 * Main service for handling business logic.
 *
 * @since 1.0
 */
@Service
public class MainService implements ServiceInterface {

    private static final Logger LOG = LoggerFactory.getLogger(MainService.class);

    private final Repository repo;

    public MainService(Repository repo) {
        this.repo = repo;
    }

    @Override
    public Optional<Entity> findById(Long id) {
        LOG.debug("Finding entity by id: {}", id);
        return repo.findById(id);
    }

    public List<Entity> findAll() {
        return repo.findAll();
    }

    public Entity save(Entity entity) {
        if (entity.getId() == null) {
            entity.setCreatedAt(Instant.now());
        }
        entity.setUpdatedAt(Instant.now());
        return repo.save(entity);
    }

    /**
     * Processes a batch of entities.
     *
     * @param entities the entities to process
     * @throws ProcessingException if processing fails
     */
    public void processBatch(List<Entity> entities) throws ProcessingException {
        for (Entity entity : entities) {
            try {
                process(entity);
            } catch (Exception e) {
                throw new ProcessingException("Failed to process: " + entity.getId(), e);
            }
        }
    }

    private void process(Entity entity) {
        // complex processing logic
        entity.setStatus(Status.PROCESSED);
        repo.save(entity);
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	rendered := output.Render()

	// Package and imports present.
	assert.Contains(t, rendered, "package com.example")
	assert.Contains(t, rendered, "import java.util.List")
	assert.Contains(t, rendered, "import java.util.Map")
	assert.Contains(t, rendered, "import java.util.Optional")

	// Class structure present.
	assert.Contains(t, rendered, "public class MainService")
	assert.Contains(t, rendered, "implements ServiceInterface")
	assert.Contains(t, rendered, "@Service")

	// Method signatures present, bodies excluded.
	assert.Contains(t, rendered, "Optional<Entity> findById(Long id)")
	assert.Contains(t, rendered, "List<Entity> findAll()")
	assert.Contains(t, rendered, "Entity save(Entity entity)")
	assert.Contains(t, rendered, "processBatch(List<Entity> entities) throws ProcessingException")

	// Bodies excluded.
	assert.NotContains(t, rendered, "LOG.debug")
	assert.NotContains(t, rendered, "entity.setCreatedAt")
	assert.NotContains(t, rendered, "complex processing logic")

	// Fields present.
	assert.Contains(t, rendered, "private static final Logger LOG")
	assert.Contains(t, rendered, "private final Repository repo")

	// Verify compression ratio.
	ratio := float64(len(rendered)) / float64(len(source))
	t.Logf("Java compression ratio: %.2f (%d -> %d bytes)", ratio, len(source), len(rendered))
	assert.Greater(t, ratio, 0.2)
	assert.Less(t, ratio, 0.75)
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestJavaCompressor_EmptyClass(t *testing.T) {
	c := NewJavaCompressor()
	source := `public class Empty {}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig)
	assert.Equal(t, "Empty", classSig.Name)
}

func TestJavaCompressor_EmptyInterface(t *testing.T) {
	c := NewJavaCompressor()
	source := `public interface Marker {}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var ifaceSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindInterface {
			ifaceSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, ifaceSig)
	assert.Equal(t, "Marker", ifaceSig.Name)
}

func TestJavaCompressor_MultipleClassesInFile(t *testing.T) {
	c := NewJavaCompressor()
	source := `class First {
    void method1() {
        // body
    }
}

class Second {
    void method2() {
        // body
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	classCount := 0
	for _, sig := range output.Signatures {
		if sig.Kind == KindClass {
			classCount++
		}
	}
	assert.Equal(t, 2, classCount, "expected 2 class signatures")
}

func TestJavaCompressor_StaticMethods(t *testing.T) {
	c := NewJavaCompressor()
	source := `public class Utils {
    public static String format(String s) {
        return s.trim().toLowerCase();
    }

    public static int parse(String s) {
        return Integer.parseInt(s);
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig)
	assert.Contains(t, classSig.Source, "public static String format(String s)")
	assert.Contains(t, classSig.Source, "public static int parse(String s)")
	assert.NotContains(t, classSig.Source, "trim().toLowerCase()")
	assert.NotContains(t, classSig.Source, "Integer.parseInt")
}

func TestJavaCompressor_OriginalSize(t *testing.T) {
	c := NewJavaCompressor()
	source := `package com.test;
public class Foo {}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	assert.Equal(t, len(source), output.OriginalSize)
	assert.Equal(t, "java", output.Language)
}

func TestJavaCompressor_LineNumbers(t *testing.T) {
	c := NewJavaCompressor()
	source := `package com.test;

import java.util.List;

public class Foo {
    void bar() {}
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	// Package should be on line 1.
	require.True(t, len(output.Signatures) > 0)
	assert.Equal(t, 1, output.Signatures[0].StartLine)

	// Class should start on line 5.
	for _, sig := range output.Signatures {
		if sig.Kind == KindClass {
			assert.Equal(t, 5, sig.StartLine)
			break
		}
	}
}