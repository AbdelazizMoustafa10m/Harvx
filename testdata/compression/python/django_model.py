"""Django model for managing blog articles."""

from django.db import models
from django.utils import timezone
from django.contrib.auth.models import User

__all__ = ["Article", "ArticleManager"]


class ArticleManager(models.Manager):
    """Custom manager for Article model."""

    def published(self):
        """Return only published articles."""
        return self.filter(status="published", pub_date__lte=timezone.now())

    def by_author(self, author):
        """Return articles by a specific author."""
        return self.filter(author=author).order_by("-pub_date")


class Article(models.Model):
    """Blog article with rich metadata."""

    STATUS_CHOICES = [
        ("draft", "Draft"),
        ("published", "Published"),
        ("archived", "Archived"),
    ]

    title: str = models.CharField(max_length=200)
    slug: str = models.SlugField(unique=True)
    author = models.ForeignKey(User, on_delete=models.CASCADE)
    body = models.TextField()
    status = models.CharField(max_length=20, choices=STATUS_CHOICES, default="draft")
    pub_date = models.DateTimeField(null=True, blank=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    objects = ArticleManager()

    class Meta:
        ordering = ["-pub_date"]
        verbose_name_plural = "articles"
        indexes = [
            models.Index(fields=["slug"]),
            models.Index(fields=["-pub_date", "status"]),
        ]

    def __str__(self) -> str:
        return self.title

    def publish(self) -> None:
        """Publish this article."""
        self.status = "published"
        self.pub_date = timezone.now()
        self.save()

    @property
    def is_published(self) -> bool:
        """Check if article is currently published."""
        return self.status == "published" and self.pub_date is not None

    def get_absolute_url(self) -> str:
        """Return the canonical URL for this article."""
        from django.urls import reverse
        return reverse("article-detail", kwargs={"slug": self.slug})
